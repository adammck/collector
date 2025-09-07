package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	pb "github.com/adammck/collector/proto/gen"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type pair struct {
	req *pb.Request
	res chan *pb.Response
}

type server struct {
	pending map[string]*pair
	pmu     sync.RWMutex

	waiters map[chan struct{}]struct{}
	wmu     sync.Mutex

	timeout time.Duration
}

func newServer() *server {
	return &server{
		pending: make(map[string]*pair),
		waiters: make(map[chan struct{}]struct{}),
		timeout: 30 * time.Second,
	}
}

func (s *server) ServeHTTP() http.Handler {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))

	mux.Handle("/", fs)
	mux.HandleFunc("/data.json", s.handleData)
	mux.HandleFunc("POST /submit/{uuid}", s.handleSubmit)

	return mux
}

type webRequest struct {
	UUID  string      `json:"uuid"`
	Proto *pb.Request `json:"proto"`
}

func (w *webRequest) MarshalJSON() ([]byte, error) {
	pj, err := protojson.Marshal(w.Proto)
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"uuid":  w.UUID,
		"proto": json.RawMessage(pj),
	})
}

// assumes s.pmu is held
func (s *server) getRandomPending() (string, *pair, bool) {
	if len(s.pending) == 0 {
		return "", nil, false
	}

	keys := make([]string, 0, len(s.pending))
	for k := range s.pending {
		keys = append(keys, k)
	}

	uuid := keys[rand.Intn(len(keys))]
	p := s.pending[uuid]

	return uuid, p, true
}

func (s *server) getPendingRequest() (string, *pair, error) {
	ch := make(chan struct{})

	s.wmu.Lock()
	s.waiters[ch] = struct{}{}
	s.wmu.Unlock()

	defer func() {
		s.wmu.Lock()
		delete(s.waiters, ch)
		s.wmu.Unlock()
	}()

	timeout := time.After(s.timeout)
	for {
		s.pmu.RLock()
		uuid, p, ok := s.getRandomPending()
		s.pmu.RUnlock()
		if ok {
			return uuid, p, nil
		}

		select {
		case <-ch:
			continue
		case <-timeout:
			return "", nil, fmt.Errorf("no pending requests after timeout")
		}
	}
}

func (s *server) handleData(w http.ResponseWriter, r *http.Request) {
	uuid, p, err := s.getPendingRequest()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := validate(p.req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	b, err := json.Marshal(webRequest{
		UUID:  uuid,
		Proto: p.req,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	u := r.PathValue("uuid")
	if u == "" {
		http.Error(w, "missing: uuid", http.StatusNotFound)
		return
	}

	s.pmu.Lock()
	p, ok := s.pending[u]
	if ok {
		delete(s.pending, u)
	}
	s.pmu.Unlock()

	if !ok {
		http.Error(w, fmt.Sprintf("pending not found: %v", u), http.StatusNotFound)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res := &pb.Response{}
	if err := protojson.Unmarshal(b, res); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.res <- res
	close(p.res)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

type collectorServer struct {
	pb.UnsafeCollectorServer
	s *server
}

func (cs *collectorServer) Collect(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	fmt.Printf("Collect: %v\n", req)

	resCh := make(chan *pb.Response, 1)
	u := uuid.NewString()
	p := &pair{
		req: req,
		res: resCh,
	}

	cs.s.pmu.Lock()
	cs.s.pending[u] = p
	cs.s.pmu.Unlock()

	// notify all waiters
	cs.s.wmu.Lock()
	for ch := range cs.s.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	cs.s.wmu.Unlock()

	select {
	case res, ok := <-resCh:
		if !ok {
			return nil, status.Error(codes.Aborted, "response channel closed")
		}
		return res, nil
	case <-ctx.Done():
		cs.s.pmu.Lock()
		delete(cs.s.pending, u)
		cs.s.pmu.Unlock()
		return nil, status.Error(codes.Canceled, "request cancelled")
	}
}

func main() {
	hp := flag.Int("http-port", 8000, "port for http server to listen on")
	gp := flag.Int("grpc-port", 50051, "port for grpc server to listen on")
	flag.Parse()

	s := newServer()

	// Start HTTP server
	go func() {
		addr := fmt.Sprintf(":%d", *hp)
		log.Printf("HTTP server listening on %s", addr)
		srv := &http.Server{
			Addr:    addr,
			Handler: s.ServeHTTP(),
		}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Start gRPC server
	go func() {
		addr := fmt.Sprintf(":%d", *gp)
		log.Printf("gRPC server listening on %s", addr)

		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		srv := grpc.NewServer()
		pb.RegisterCollectorServer(srv, &collectorServer{s: s})

		if err := srv.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	select {}
}

func validate(req *pb.Request) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	// Add your validation logic here
	return nil
}
