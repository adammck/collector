package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/adammck/collector/proto/gen"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMain(m *testing.M) {
	m.Run()
}

// test utilities

func newTestServer() *server {
	return newServer()
}

func newTestRequest() *pb.Request {
	return &pb.Request{
		Inputs: []*pb.Input{
			{
				Visualization: &pb.Input_Grid{
					Grid: &pb.Grid{Rows: 10, Cols: 10},
				},
				Data: &pb.Data{
					Data: &pb.Data_Ints{
						Ints: &pb.Ints{Values: make([]int64, 100)}, // 10x10 = 100
					},
				},
			},
		},
		Output: &pb.OutputSchema{
			Output: &pb.OutputSchema_OptionList{
				OptionList: &pb.OptionListSchema{
					Options: []*pb.Option{
						{Label: "Option 1", Hotkey: "1"},
						{Label: "Option 2", Hotkey: "2"},
					},
				},
			},
		},
	}
}

func newTestResponse() *pb.Response {
	return &pb.Response{
		Output: &pb.Output{
			Output: &pb.Output_OptionList{
				OptionList: &pb.OptionListOutput{Index: 0},
			},
		},
	}
}

// grpc test helper
func startTestGRPCServer(t *testing.T, s *server) (pb.CollectorClient, func()) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterCollectorServer(srv, &collectorServer{s: s})

	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Logf("grpc server error: %v", err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	client := pb.NewCollectorClient(conn)

	cleanup := func() {
		conn.Close()
		srv.Stop()
		lis.Close()
	}

	return client, cleanup
}

// queue integration tests

func TestServerQueueIntegration(t *testing.T) {
	s := newTestServer()

	// test queue is initially empty
	status := s.queue.Status()
	if status.Total != 0 {
		t.Fatalf("expected empty queue, got total %d", status.Total)
	}

	// create queue item
	req := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := "test-uuid"
	item := &QueueItem{
		ID:       testUUID,
		Request:  req,
		Response: resCh,
		AddedAt:  time.Now(),
		Context:  context.Background(),
	}

	// enqueue item
	err := s.queue.Enqueue(item)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// verify queue status
	status = s.queue.Status()
	if status.Total != 1 {
		t.Fatalf("expected queue total 1, got %d", status.Total)
	}
	if status.Active != 1 {
		t.Fatalf("expected queue active 1, got %d", status.Active)
	}

	// test moving item to current
	s.cmu.Lock()
	s.current[testUUID] = item
	s.cmu.Unlock()

	// verify item is in current
	s.cmu.RLock()
	currentItem, exists := s.current[testUUID]
	s.cmu.RUnlock()

	if !exists {
		t.Fatal("expected item to be in current map")
	}
	if currentItem.Request != req {
		t.Fatal("expected request to match")
	}
	if currentItem.Response != resCh {
		t.Fatal("expected response channel to match")
	}

	// test deletion from current
	s.cmu.Lock()
	delete(s.current, testUUID)
	s.cmu.Unlock()

	s.cmu.RLock()
	_, exists = s.current[testUUID]
	s.cmu.RUnlock()

	if exists {
		t.Fatal("expected current item to be deleted")
	}
}

func TestConcurrentCurrentAccess(t *testing.T) {
	s := newTestServer()
	const numGoroutines = 10
	const numOperations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// concurrent operations on current map
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				uuid := fmt.Sprintf("uuid-%d-%d", id, j)
				req := newTestRequest()
				resCh := make(chan *pb.Response, 1)
				item := &QueueItem{
					ID:       uuid,
					Request:  req,
					Response: resCh,
					AddedAt:  time.Now(),
					Context:  context.Background(),
				}

				// add to current
				s.cmu.Lock()
				s.current[uuid] = item
				s.cmu.Unlock()

				// sometimes delete immediately
				if j%2 == 0 {
					s.cmu.Lock()
					delete(s.current, uuid)
					s.cmu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
}

// http handler tests

func TestHandleDataNoPending(t *testing.T) {
	s := newTestServer()
	s.timeout = 100 * time.Millisecond // short timeout for test
	req := httptest.NewRequest("GET", "/data.json", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	s.handleData(w, req)
	duration := time.Since(start)

	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("expected status 408, got %d", w.Code)
	}

	// should have waited at least the timeout duration
	if duration < 100*time.Millisecond {
		t.Fatalf("expected at least 100ms delay, got %v", duration)
	}

	expectedMsg := "no pending requests available"
	if !strings.Contains(w.Body.String(), expectedMsg) {
		t.Fatalf("expected timeout message, got: %s", w.Body.String())
	}
}

func TestHandleDataWithPending(t *testing.T) {
	s := newTestServer()

	// add item to queue
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := uuid.NewString()
	item := &QueueItem{
		ID:       testUUID,
		Request:  testReq,
		Response: resCh,
		AddedAt:  time.Now(),
		Context:  context.Background(),
	}

	s.queue.Enqueue(item)

	req := httptest.NewRequest("GET", "/data.json", nil)
	w := httptest.NewRecorder()

	s.handleData(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected json content type, got: %s", contentType)
	}

	// parse the json response manually since webRequest has custom marshaling
	var jsonResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	webReqUUID, ok := jsonResp["uuid"].(string)
	if !ok {
		t.Fatal("expected uuid to be a string")
	}

	protoData, ok := jsonResp["proto"].(map[string]interface{})
	if !ok {
		t.Fatal("expected proto to be an object")
	}

	if webReqUUID != testUUID {
		t.Fatalf("expected uuid %s, got %s", testUUID, webReqUUID)
	}

	// verify proto has expected structure
	inputs, ok := protoData["inputs"].([]interface{})
	if !ok || len(inputs) == 0 {
		t.Fatal("expected proto inputs to be non-empty array")
	}

	// verify the item is now in current map
	s.cmu.RLock()
	currentItem, exists := s.current[testUUID]
	s.cmu.RUnlock()

	if !exists {
		t.Fatal("expected item to be in current map after handleData")
	}
	if currentItem.Request != testReq {
		t.Fatal("expected current item request to match")
	}
}

func TestHandleSubmitValid(t *testing.T) {
	s := newTestServer()

	// create queue item and add to current (simulate handleData)
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := uuid.NewString()
	item := &QueueItem{
		ID:       testUUID,
		Request:  testReq,
		Response: resCh,
		AddedAt:  time.Now(),
		Context:  context.Background(),
	}

	// add to current (simulate what handleData would do)
	s.cmu.Lock()
	s.current[testUUID] = item
	s.cmu.Unlock()

	// prepare response
	testRes := newTestResponse()
	resJSON, err := protojson.Marshal(testRes)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	req := httptest.NewRequest("POST", "/submit/"+testUUID, bytes.NewReader(resJSON))
	w := httptest.NewRecorder()

	// simulate the path value extraction (normally done by http.ServeMux)
	req.SetPathValue("uuid", testUUID)

	s.handleSubmit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// verify response channel received the data
	select {
	case res := <-resCh:
		if res == nil {
			t.Fatal("expected response to be non-nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected response on channel")
	}

	// verify current request was removed
	s.cmu.RLock()
	_, exists := s.current[testUUID]
	s.cmu.RUnlock()

	if exists {
		t.Fatal("expected current request to be removed")
	}
}

func TestHandleSubmitInvalidUUID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest("POST", "/submit/invalid-uuid", nil)
	w := httptest.NewRecorder()

	req.SetPathValue("uuid", "invalid-uuid")

	s.handleSubmit(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	expectedMsg := "pending request not found"
	if !strings.Contains(w.Body.String(), expectedMsg) {
		t.Fatalf("expected 'pending request not found' message, got: %s", w.Body.String())
	}
}

func TestHandleSubmitMalformedJSON(t *testing.T) {
	s := newTestServer()

	// create queue item and add to current
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := uuid.NewString()
	item := &QueueItem{
		ID:       testUUID,
		Request:  testReq,
		Response: resCh,
		AddedAt:  time.Now(),
		Context:  context.Background(),
	}

	s.cmu.Lock()
	s.current[testUUID] = item
	s.cmu.Unlock()

	malformedJSON := []byte(`{"invalid": json`)
	req := httptest.NewRequest("POST", "/submit/"+testUUID, bytes.NewReader(malformedJSON))
	w := httptest.NewRecorder()

	req.SetPathValue("uuid", testUUID)

	s.handleSubmit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// grpc service tests

func TestCollectorServiceSuccess(t *testing.T) {
	s := newTestServer()
	client, cleanup := startTestGRPCServer(t, s)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testReq := newTestRequest()

	// make grpc call in background
	resultCh := make(chan *pb.Response, 1)
	errCh := make(chan error, 1)

	go func() {
		res, err := client.Collect(ctx, testReq)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- res
	}()

	// wait a bit for request to be registered
	time.Sleep(10 * time.Millisecond)

	// check that queue has item
	status := s.queue.Status()
	if status.Total != 1 {
		t.Fatalf("expected 1 item in queue, got %d", status.Total)
	}

	// get item from queue (simulating web client)
	item, err := s.queue.GetNext(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("failed to get item from queue: %v", err)
	}

	// add to current (simulating handleData)
	s.cmu.Lock()
	s.current[item.ID] = item
	s.cmu.Unlock()

	// simulate web client submission
	testRes := newTestResponse()
	resJSON, err := protojson.Marshal(testRes)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// simulate http submission
	req := httptest.NewRequest("POST", "/submit/"+item.ID, bytes.NewReader(resJSON))
	w := httptest.NewRecorder()
	req.SetPathValue("uuid", item.ID)

	s.handleSubmit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("submit failed: %d: %s", w.Code, w.Body.String())
	}

	// verify grpc call completed successfully
	select {
	case res := <-resultCh:
		if res == nil {
			t.Fatal("expected non-nil response")
		}
	case err := <-errCh:
		t.Fatalf("grpc call failed: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("grpc call timed out")
	}
}

func TestCollectorServiceCancellation(t *testing.T) {
	s := newTestServer()
	client, cleanup := startTestGRPCServer(t, s)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	testReq := newTestRequest()

	// make grpc call in background
	errCh := make(chan error, 1)

	go func() {
		_, err := client.Collect(ctx, testReq)
		errCh <- err
	}()

	// wait for request to be registered
	time.Sleep(10 * time.Millisecond)

	// verify queue has item
	status := s.queue.Status()
	if status.Total != 1 {
		t.Fatalf("expected 1 item in queue, got %d", status.Total)
	}

	// cancel context
	cancel()

	// verify grpc call was cancelled
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
		// should be a grpc cancellation error
	case <-time.After(1 * time.Second):
		t.Fatal("expected cancellation error")
	}

	// wait a bit for cleanup
	time.Sleep(10 * time.Millisecond)

	// verify queue was cleaned up (context cancellation removes items)
	status = s.queue.Status()
	if status.Total != 0 {
		t.Fatalf("expected 0 items in queue after cancellation, got %d", status.Total)
	}
}

// integration tests

func TestEndToEndFlow(t *testing.T) {
	s := newTestServer()
	client, cleanup := startTestGRPCServer(t, s)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testReq := newTestRequest()

	// start grpc call
	resultCh := make(chan *pb.Response, 1)
	errCh := make(chan error, 1)

	go func() {
		res, err := client.Collect(ctx, testReq)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- res
	}()

	// wait for request registration
	time.Sleep(10 * time.Millisecond)

	// simulate web client getting data
	httpReq := httptest.NewRequest("GET", "/data.json", nil)
	w := httptest.NewRecorder()

	s.handleData(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("data request failed: %d: %s", w.Code, w.Body.String())
	}

	// parse the json response manually since webRequest has custom marshaling
	var jsonResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonResp); err != nil {
		t.Fatalf("failed to unmarshal web request: %v", err)
	}

	webReqUUID, ok := jsonResp["uuid"].(string)
	if !ok {
		t.Fatal("expected uuid to be a string")
	}

	protoData, ok := jsonResp["proto"].(map[string]interface{})
	if !ok {
		t.Fatal("expected proto to be an object")
	}

	// verify the request has expected structure
	inputs, ok := protoData["inputs"].([]interface{})
	if !ok || len(inputs) != len(testReq.Inputs) {
		t.Fatalf("expected %d proto inputs, got %v", len(testReq.Inputs), len(inputs))
	}

	// simulate web client submission
	testRes := newTestResponse()
	resJSON, err := protojson.Marshal(testRes)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	submitReq := httptest.NewRequest("POST", "/submit/"+webReqUUID, bytes.NewReader(resJSON))
	submitW := httptest.NewRecorder()
	submitReq.SetPathValue("uuid", webReqUUID)

	s.handleSubmit(submitW, submitReq)

	if submitW.Code != http.StatusOK {
		t.Fatalf("submit failed: %d: %s", submitW.Code, submitW.Body.String())
	}

	// verify grpc call completed
	select {
	case res := <-resultCh:
		if res == nil {
			t.Fatal("expected non-nil response")
		}
		// verify response content matches
		if res.Output == nil {
			t.Fatal("expected output to be non-nil")
		}
	case err := <-errCh:
		t.Fatalf("grpc call failed: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("end-to-end flow timed out")
	}
}

// additional coverage tests

func TestServeHTTP(t *testing.T) {
	s := newTestServer()
	handler := s.ServeHTTP()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// test that static files are served
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// should get some response (likely 404 since no static files exist)
	if w.Code == 0 {
		t.Fatal("expected some response code")
	}
}

func TestWebRequestMarshalJSON(t *testing.T) {
	testReq := newTestRequest()
	webReq := webRequest{
		UUID:  "test-uuid",
		Proto: testReq,
		Queue: QueueStatus{Total: 1, Active: 1, Deferred: 0},
	}

	data, err := json.Marshal(webReq)
	if err != nil {
		t.Fatalf("failed to marshal webRequest: %v", err)
	}

	// verify it produces json with uuid, proto, and queue fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result["uuid"] != "test-uuid" {
		t.Fatalf("expected uuid field, got: %v", result["uuid"])
	}

	if result["proto"] == nil {
		t.Fatal("expected proto field")
	}

	if result["queue"] == nil {
		t.Fatal("expected queue field")
	}
}

func TestHandleQueueStatus(t *testing.T) {
	s := newTestServer()

	// add some items to queue
	for i := 0; i < 3; i++ {
		item := &QueueItem{
			ID:       fmt.Sprintf("test-%d", i),
			Request:  newTestRequest(),
			Response: make(chan *pb.Response, 1),
			AddedAt:  time.Now(),
			Context:  context.Background(),
		}
		s.queue.Enqueue(item)
	}

	req := httptest.NewRequest("GET", "/queue/status", nil)
	w := httptest.NewRecorder()

	s.handleQueueStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var status QueueStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to unmarshal status: %v", err)
	}

	if status.Total != 3 {
		t.Fatalf("expected total 3, got %d", status.Total)
	}
	if status.Active != 3 {
		t.Fatalf("expected active 3, got %d", status.Active)
	}
}

// errorReader always returns an error when read
type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

func TestHandleSubmitReadError(t *testing.T) {
	s := newTestServer()

	// create queue item and add to current
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := "test-uuid"
	item := &QueueItem{
		ID:       testUUID,
		Request:  testReq,
		Response: resCh,
		AddedAt:  time.Now(),
		Context:  context.Background(),
	}

	s.cmu.Lock()
	s.current[testUUID] = item
	s.cmu.Unlock()

	// create a request with an error reader
	errorReader := &errorReader{}
	req := httptest.NewRequest("POST", "/submit/"+testUUID, errorReader)
	w := httptest.NewRecorder()

	req.SetPathValue("uuid", testUUID)

	s.handleSubmit(w, req)

	// should get an internal server error due to read failure
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for read error, got %d", w.Code)
	}
}

// validation tests

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.Request
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errMsg:  "request cannot be nil",
		},
		{
			name: "empty inputs",
			req: &pb.Request{
				Inputs: []*pb.Input{},
				Output: &pb.OutputSchema{
					Output: &pb.OutputSchema_OptionList{
						OptionList: &pb.OptionListSchema{
							Options: []*pb.Option{
								{Label: "Option 1", Hotkey: "1"},
								{Label: "Option 2", Hotkey: "2"},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "request must have at least one input",
		},
		{
			name:    "valid request",
			req:     newTestRequest(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidateInput(t *testing.T) {
	validData := &pb.Data{
		Data: &pb.Data_Ints{
			Ints: &pb.Ints{Values: make([]int64, 100)}, // 10x10 = 100
		},
	}

	tests := []struct {
		name    string
		input   *pb.Input
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
			errMsg:  "input cannot be nil",
		},
		{
			name: "nil visualization",
			input: &pb.Input{
				Visualization: nil,
				Data:          validData,
			},
			wantErr: true,
			errMsg:  "visualization is required",
		},
		{
			name: "valid input",
			input: &pb.Input{
				Visualization: &pb.Input_Grid{
					Grid: &pb.Grid{Rows: 10, Cols: 10},
				},
				Data: validData,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInput(tt.input, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidateGrid(t *testing.T) {
	tests := []struct {
		name    string
		grid    *pb.Grid
		data    *pb.Data
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil grid",
			grid:    nil,
			data:    &pb.Data{},
			wantErr: true,
			errMsg:  "grid cannot be nil",
		},
		{
			name:    "zero rows",
			grid:    &pb.Grid{Rows: 0, Cols: 5},
			data:    &pb.Data{},
			wantErr: true,
			errMsg:  "grid dimensions must be positive",
		},
		{
			name:    "zero cols",
			grid:    &pb.Grid{Rows: 5, Cols: 0},
			data:    &pb.Data{},
			wantErr: true,
			errMsg:  "grid dimensions must be positive",
		},
		{
			name:    "negative rows",
			grid:    &pb.Grid{Rows: -1, Cols: 5},
			data:    &pb.Data{},
			wantErr: true,
			errMsg:  "grid dimensions must be positive",
		},
		{
			name:    "too large grid",
			grid:    &pb.Grid{Rows: 101, Cols: 50},
			data:    &pb.Data{},
			wantErr: true,
			errMsg:  "grid too large",
		},
		{
			name:    "nil data",
			grid:    &pb.Grid{Rows: 2, Cols: 2},
			data:    nil,
			wantErr: true,
			errMsg:  "data is required",
		},
		{
			name:    "nil data type",
			grid:    &pb.Grid{Rows: 2, Cols: 2},
			data:    &pb.Data{Data: nil},
			wantErr: true,
			errMsg:  "data type is required",
		},
		{
			name: "nil ints data",
			grid: &pb.Grid{Rows: 2, Cols: 2},
			data: &pb.Data{
				Data: &pb.Data_Ints{Ints: nil},
			},
			wantErr: true,
			errMsg:  "ints data cannot be nil",
		},
		{
			name: "wrong ints size",
			grid: &pb.Grid{Rows: 2, Cols: 2},
			data: &pb.Data{
				Data: &pb.Data_Ints{
					Ints: &pb.Ints{Values: []int64{1, 2, 3}}, // should be 4
				},
			},
			wantErr: true,
			errMsg:  "data size 3 doesn't match grid size 4",
		},
		{
			name: "nil floats data",
			grid: &pb.Grid{Rows: 2, Cols: 2},
			data: &pb.Data{
				Data: &pb.Data_Floats{Floats: nil},
			},
			wantErr: true,
			errMsg:  "floats data cannot be nil",
		},
		{
			name: "wrong floats size",
			grid: &pb.Grid{Rows: 2, Cols: 2},
			data: &pb.Data{
				Data: &pb.Data_Floats{
					Floats: &pb.Floats{Values: []float64{1.0, 2.0, 3.0}}, // should be 4
				},
			},
			wantErr: true,
			errMsg:  "data size 3 doesn't match grid size 4",
		},
		{
			name: "valid ints",
			grid: &pb.Grid{Rows: 2, Cols: 2},
			data: &pb.Data{
				Data: &pb.Data_Ints{
					Ints: &pb.Ints{Values: []int64{1, 2, 3, 4}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid floats",
			grid: &pb.Grid{Rows: 2, Cols: 2},
			data: &pb.Data{
				Data: &pb.Data_Floats{
					Floats: &pb.Floats{Values: []float64{1.0, 2.0, 3.0, 4.0}},
				},
			},
			wantErr: false,
		},
		{
			name: "1x1 grid valid",
			grid: &pb.Grid{Rows: 1, Cols: 1},
			data: &pb.Data{
				Data: &pb.Data_Ints{
					Ints: &pb.Ints{Values: []int64{42}},
				},
			},
			wantErr: false,
		},
		{
			name: "max size grid valid",
			grid: &pb.Grid{Rows: 100, Cols: 100},
			data: &pb.Data{
				Data: &pb.Data_Ints{
					Ints: &pb.Ints{Values: make([]int64, 10000)},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGrid(tt.grid, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGrid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidateData(t *testing.T) {
	tests := []struct {
		name    string
		data    *pb.Data
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
			errMsg:  "data cannot be nil",
		},
		{
			name:    "nil data type",
			data:    &pb.Data{Data: nil},
			wantErr: true,
			errMsg:  "data type is required",
		},
		{
			name: "valid ints",
			data: &pb.Data{
				Data: &pb.Data_Ints{
					Ints: &pb.Ints{Values: []int64{1, 2, 3}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid floats",
			data: &pb.Data{
				Data: &pb.Data_Floats{
					Floats: &pb.Floats{Values: []float64{1.0, 2.0, 3.0}},
				},
			},
			wantErr: false,
		},
		{
			name: "nil floats data",
			data: &pb.Data{
				Data: &pb.Data_Floats{Floats: nil},
			},
			wantErr: true,
			errMsg:  "floats data cannot be nil",
		},
		{
			name: "nan float",
			data: &pb.Data{
				Data: &pb.Data_Floats{
					Floats: &pb.Floats{Values: []float64{1.0, math.NaN(), 3.0}},
				},
			},
			wantErr: true,
			errMsg:  "float value at index 1 is NaN",
		},
		{
			name: "positive inf float",
			data: &pb.Data{
				Data: &pb.Data_Floats{
					Floats: &pb.Floats{Values: []float64{1.0, math.Inf(1), 3.0}},
				},
			},
			wantErr: true,
			errMsg:  "float value at index 1 is infinite",
		},
		{
			name: "negative inf float",
			data: &pb.Data{
				Data: &pb.Data_Floats{
					Floats: &pb.Floats{Values: []float64{1.0, math.Inf(-1), 3.0}},
				},
			},
			wantErr: true,
			errMsg:  "float value at index 1 is infinite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidateOutputSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  *pb.OutputSchema
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil schema",
			schema:  nil,
			wantErr: true,
			errMsg:  "output schema is required",
		},
		{
			name:    "nil output type",
			schema:  &pb.OutputSchema{Output: nil},
			wantErr: true,
			errMsg:  "output type is required",
		},
		{
			name: "nil option list",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{OptionList: nil},
			},
			wantErr: true,
			errMsg:  "option list cannot be nil",
		},
		{
			name: "empty option list",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{Options: []*pb.Option{}},
				},
			},
			wantErr: true,
			errMsg:  "option list must have at least 2 options",
		},
		{
			name: "one option only",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "Option 1", Hotkey: "1"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "option list must have at least 2 options",
		},
		{
			name: "nil option",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "Option 1", Hotkey: "1"},
							nil,
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "option 1 cannot be nil",
		},
		{
			name: "empty label",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "Option 1", Hotkey: "1"},
							{Label: "", Hotkey: "2"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "option 1 label cannot be empty",
		},
		{
			name: "empty hotkey",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "Option 1", Hotkey: "1"},
							{Label: "Option 2", Hotkey: ""},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "option 1 hotkey must be single character",
		},
		{
			name: "multi-char hotkey",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "Option 1", Hotkey: "1"},
							{Label: "Option 2", Hotkey: "ab"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "option 1 hotkey must be single character",
		},
		{
			name: "duplicate hotkey",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "Option 1", Hotkey: "1"},
							{Label: "Option 2", Hotkey: "1"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate hotkey \"1\" found at option 1",
		},
		{
			name: "valid option list",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "Option 1", Hotkey: "1"},
							{Label: "Option 2", Hotkey: "2"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid many options",
			schema: &pb.OutputSchema{
				Output: &pb.OutputSchema_OptionList{
					OptionList: &pb.OptionListSchema{
						Options: []*pb.Option{
							{Label: "First", Hotkey: "a"},
							{Label: "Second", Hotkey: "b"},
							{Label: "Third", Hotkey: "c"},
							{Label: "Fourth", Hotkey: "d"},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputSchema(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOutputSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestCollectorValidationFailure(t *testing.T) {
	s := newTestServer()
	client, cleanup := startTestGRPCServer(t, s)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// create invalid request
	badReq := &pb.Request{
		Inputs: []*pb.Input{}, // empty inputs should fail validation
		Output: &pb.OutputSchema{
			Output: &pb.OutputSchema_OptionList{
				OptionList: &pb.OptionListSchema{
					Options: []*pb.Option{
						{Label: "Option 1", Hotkey: "1"},
						{Label: "Option 2", Hotkey: "2"},
					},
				},
			},
		},
	}

	_, err := client.Collect(ctx, badReq)
	if err == nil {
		t.Fatal("expected validation error")
	}

	// should be grpc invalid argument error
	if !strings.Contains(err.Error(), "request must have at least one input") {
		t.Fatalf("expected validation error message, got: %v", err)
	}
}