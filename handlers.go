package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	pb "github.com/adammck/collector/proto/gen"
	"google.golang.org/protobuf/encoding/protojson"
)

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

	// Remove from current before deferring
	s.cmu.Lock()
	delete(s.current, u)
	s.cmu.Unlock()

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

func (s *server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats := getStats()
	queueStatus := s.queue.Status()
	
	metrics := map[string]interface{}{
		"queue": queueStatus,
		"errors": map[string]int64{
			"validation": stats.ValidationErrors,
			"timeout": stats.TimeoutErrors,
			"internal": stats.InternalErrors,
			"resource_exhausted": stats.ResourceExhausted,
		},
		"total_requests": stats.TotalRequests,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"queue_total": s.queue.Status().Total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *server) ServeHTTP() http.Handler {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./frontend/dist"))

	mux.Handle("/", fs)
	mux.HandleFunc("/data.json", s.handleData)
	mux.HandleFunc("POST /submit/{uuid}", s.handleSubmit)
	mux.HandleFunc("POST /defer/{uuid}", s.handleDefer)
	mux.HandleFunc("GET /queue/status", s.handleQueueStatus)
	mux.HandleFunc("GET /metrics", s.handleMetrics)
	mux.HandleFunc("GET /health", s.handleHealth)

	return mux
}