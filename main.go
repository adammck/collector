package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

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
	proxy   chan *pair
	pending map[string]*pair
	mu      sync.RWMutex
}

func newServer() *server {
	return &server{
		proxy:   make(chan *pair),
		pending: make(map[string]*pair),
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

func (s *server) handleData(w http.ResponseWriter, r *http.Request) {
	// Block until the next request (from gRPC) is available
	p := <-s.proxy

	// Store the response chan for later
	u := uuid.NewString()
	s.mu.Lock()
	s.pending[u] = p
	s.mu.Unlock()

	if err := validate(p.req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	b, err := protojson.Marshal(p.req)
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

	s.mu.Lock()
	p, ok := s.pending[u]
	if ok {
		delete(s.pending, u)
	}
	s.mu.Unlock()

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
	resCh := make(chan *pb.Response, 1)
	p := &pair{
		req: req,
		res: resCh,
	}

	// Send request to HTTP server
	select {
	case cs.s.proxy <- p:
		// Wait for response
		select {
		case res, ok := <-resCh:
			if !ok {
				return nil, status.Error(codes.Aborted, "response channel closed")
			}
			return res, nil
		case <-ctx.Done():
			return nil, status.Error(codes.Canceled, "request cancelled")
		}
	case <-ctx.Done():
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
