package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
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

const (
	maxPendingRequests = 1000
	httpTimeout        = 30 * time.Second
	submitTimeout      = 5 * time.Second
)

var errTimeout = errors.New("no pending requests after timeout")

type pair struct {
	req *pb.Request
	res chan *pb.Response
}

type server struct {
	queue   *Queue
	current map[string]*QueueItem
	cmu     sync.RWMutex

	timeout time.Duration
}

func newServer() *server {
	return &server{
		queue:   NewQueue(),
		current: make(map[string]*QueueItem),
		timeout: 30 * time.Second,
	}
}

func (s *server) ServeHTTP() http.Handler {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))

	mux.Handle("/", fs)
	mux.HandleFunc("/data.json", s.handleData)
	mux.HandleFunc("POST /submit/{uuid}", s.handleSubmit)
	mux.HandleFunc("POST /defer/{uuid}", s.handleDefer)
	mux.HandleFunc("GET /queue/status", s.handleQueueStatus)

	return mux
}

type webRequest struct {
	UUID  string      `json:"uuid"`
	Proto *pb.Request `json:"proto"`
	Queue QueueStatus `json:"queue"`
}

func (w *webRequest) MarshalJSON() ([]byte, error) {
	pj, err := protojson.Marshal(w.Proto)
	if err != nil {
		return nil, err
	}

	return json.Marshal(map[string]interface{}{
		"uuid":  w.UUID,
		"proto": json.RawMessage(pj),
		"queue": w.Queue,
	})
}

func (s *server) handleData(w http.ResponseWriter, r *http.Request) {
	item, err := s.queue.GetNext(s.timeout)
	if err != nil {
		writeJSONError(w, http.StatusRequestTimeout,
			"no pending requests available",
			"wait and retry")
		return
	}

	if err := validate(item.Request); err != nil {
		writeJSONError(w, http.StatusBadRequest,
			"invalid request data",
			err.Error())
		return
	}

	s.cmu.Lock()
	s.current[item.ID] = item
	s.cmu.Unlock()

	status := s.queue.Status()

	b, err := json.Marshal(webRequest{
		UUID:  item.ID,
		Proto: item.Request,
		Queue: status,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError,
			"failed to marshal request",
			err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	u := r.PathValue("uuid")
	if u == "" {
		writeJSONError(w, http.StatusBadRequest,
			"missing uuid parameter")
		return
	}

	s.cmu.Lock()
	item, ok := s.current[u]
	if ok {
		delete(s.current, u)
	}
	s.cmu.Unlock()

	if !ok {
		writeJSONError(w, http.StatusNotFound,
			"pending request not found",
			fmt.Sprintf("uuid: %s", u))
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError,
			"failed to read request body",
			err.Error())
		return
	}

	res := &pb.Response{}
	if err := protojson.Unmarshal(b, res); err != nil {
		writeJSONError(w, http.StatusBadRequest,
			"invalid response format",
			err.Error())
		return
	}

	item.Response <- res
	close(item.Response)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *server) handleDefer(w http.ResponseWriter, r *http.Request) {
	u := r.PathValue("uuid")
	if u == "" {
		writeJSONError(w, http.StatusBadRequest,
			"missing uuid parameter")
		return
	}

	if err := s.queue.Defer(u); err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	// immediately serve next item
	s.handleData(w, r)
}

func (s *server) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	status := s.queue.Status()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

type collectorServer struct {
	pb.UnsafeCollectorServer
	s *server
}

func (cs *collectorServer) Collect(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	fmt.Printf("Collect: %v\n", req)

	// validate first
	if err := validate(req); err != nil {
		return nil, validationError("invalid request: %v", err)
	}

	// check resource limits
	queueStatus := cs.s.queue.Status()
	if queueStatus.Total >= maxPendingRequests {
		return nil, resourceExhaustedError("pending requests")
	}

	resCh := make(chan *pb.Response, 1)
	u := uuid.NewString()
	item := &QueueItem{
		ID:       u,
		Request:  req,
		Response: resCh,
		AddedAt:  time.Now(),
		Context:  ctx,
	}

	if err := cs.s.queue.Enqueue(item); err != nil {
		return nil, internalError(err)
	}

	// cleanup on all exit paths
	defer func() {
		cs.s.queue.Remove(u)
	}()

	select {
	case res, ok := <-resCh:
		if !ok {
			return nil, internalError(fmt.Errorf("response channel closed"))
		}
		return res, nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, timeoutError("collect")
		}
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

	if len(req.Inputs) == 0 {
		return fmt.Errorf("request must have at least one input")
	}

	for i, input := range req.Inputs {
		if err := validateInput(input, i); err != nil {
			return fmt.Errorf("input %d: %w", i, err)
		}
	}

	if err := validateOutputSchema(req.Output); err != nil {
		return fmt.Errorf("output schema: %w", err)
	}

	return nil
}

func validateInput(input *pb.Input, index int) error {
	if input == nil {
		return fmt.Errorf("input cannot be nil")
	}

	switch v := input.Visualization.(type) {
	case *pb.Input_Grid:
		if err := validateGrid(v.Grid, input.Data); err != nil {
			return err
		}
	case nil:
		return fmt.Errorf("visualization is required")
	default:
		return fmt.Errorf("unsupported visualization type")
	}

	return validateData(input.Data)
}

func validateGrid(grid *pb.Grid, data *pb.Data) error {
	if grid == nil {
		return fmt.Errorf("grid cannot be nil")
	}

	if grid.Rows <= 0 || grid.Cols <= 0 {
		return fmt.Errorf("grid dimensions must be positive (got %dx%d)", grid.Rows, grid.Cols)
	}

	if grid.Rows > 100 || grid.Cols > 100 {
		return fmt.Errorf("grid too large (max 100x100, got %dx%d)", grid.Rows, grid.Cols)
	}

	if data == nil {
		return fmt.Errorf("data is required")
	}

	expectedSize := int(grid.Rows * grid.Cols)

	switch d := data.Data.(type) {
	case *pb.Data_Ints:
		if d.Ints == nil {
			return fmt.Errorf("ints data cannot be nil")
		}
		if len(d.Ints.Values) != expectedSize {
			return fmt.Errorf("data size %d doesn't match grid size %d", len(d.Ints.Values), expectedSize)
		}
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		if len(d.Floats.Values) != expectedSize {
			return fmt.Errorf("data size %d doesn't match grid size %d", len(d.Floats.Values), expectedSize)
		}
	case nil:
		return fmt.Errorf("data type is required")
	default:
		return fmt.Errorf("unsupported data type")
	}

	return nil
}

func validateData(data *pb.Data) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	switch d := data.Data.(type) {
	case *pb.Data_Ints:
		return nil
	case *pb.Data_Floats:
		if d.Floats == nil {
			return fmt.Errorf("floats data cannot be nil")
		}
		for i, v := range d.Floats.Values {
			if math.IsNaN(v) {
				return fmt.Errorf("float value at index %d is NaN", i)
			}
			if math.IsInf(v, 0) {
				return fmt.Errorf("float value at index %d is infinite", i)
			}
		}
		return nil
	case nil:
		return fmt.Errorf("data type is required")
	default:
		return fmt.Errorf("unsupported data type")
	}
}

func validateOutputSchema(schema *pb.OutputSchema) error {
	if schema == nil {
		return fmt.Errorf("output schema is required")
	}

	switch s := schema.Output.(type) {
	case *pb.OutputSchema_OptionList:
		if s.OptionList == nil {
			return fmt.Errorf("option list cannot be nil")
		}
		if len(s.OptionList.Options) < 2 {
			return fmt.Errorf("option list must have at least 2 options (got %d)", len(s.OptionList.Options))
		}

		hotkeys := make(map[string]bool)
		for i, opt := range s.OptionList.Options {
			if opt == nil {
				return fmt.Errorf("option %d cannot be nil", i)
			}
			if opt.Label == "" {
				return fmt.Errorf("option %d label cannot be empty", i)
			}
			if len(opt.Hotkey) != 1 {
				return fmt.Errorf("option %d hotkey must be single character (got %q)", i, opt.Hotkey)
			}
			if hotkeys[opt.Hotkey] {
				return fmt.Errorf("duplicate hotkey %q found at option %d", opt.Hotkey, i)
			}
			hotkeys[opt.Hotkey] = true
		}
		return nil
	case nil:
		return fmt.Errorf("output type is required")
	default:
		return fmt.Errorf("unsupported output schema type")
	}
}
