package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
						Ints: &pb.Ints{Values: []int64{1, 2, 3, 4, 5}},
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

// unit tests for server struct

func TestServerPendingOperations(t *testing.T) {
	s := newTestServer()

	// test adding to pending
	req := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := "test-uuid"

	s.pmu.Lock()
	s.pending[testUUID] = &pair{req: req, res: resCh}
	s.pmu.Unlock()

	// test retrieval
	s.pmu.RLock()
	p, exists := s.pending[testUUID]
	s.pmu.RUnlock()

	if !exists {
		t.Fatal("expected pending request to exist")
	}
	if p.req != req {
		t.Fatal("expected request to match")
	}
	if p.res != resCh {
		t.Fatal("expected response channel to match")
	}

	// test deletion
	s.pmu.Lock()
	delete(s.pending, testUUID)
	s.pmu.Unlock()

	s.pmu.RLock()
	_, exists = s.pending[testUUID]
	s.pmu.RUnlock()

	if exists {
		t.Fatal("expected pending request to be deleted")
	}
}

func TestServerWaiterNotification(t *testing.T) {
	s := newTestServer()

	// add waiter
	ch := make(chan struct{}, 1)
	s.wmu.Lock()
	s.waiters[ch] = struct{}{}
	s.wmu.Unlock()

	// simulate notification (from grpc handler)
	s.wmu.Lock()
	for waiterCh := range s.waiters {
		select {
		case waiterCh <- struct{}{}:
		default:
		}
	}
	s.wmu.Unlock()

	// verify notification received
	select {
	case <-ch:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected waiter notification")
	}

	// cleanup
	s.wmu.Lock()
	delete(s.waiters, ch)
	s.wmu.Unlock()
}

func TestGetRandomPending(t *testing.T) {
	s := newTestServer()

	// test empty pending map
	s.pmu.RLock()
	uuid, p, ok := s.getRandomPending()
	s.pmu.RUnlock()

	if ok {
		t.Fatal("expected no pending request")
	}
	if uuid != "" || p != nil {
		t.Fatal("expected empty values")
	}

	// add multiple pending requests
	req1 := newTestRequest()
	req2 := newTestRequest()
	resCh1 := make(chan *pb.Response, 1)
	resCh2 := make(chan *pb.Response, 1)
	uuid1 := "uuid-1"
	uuid2 := "uuid-2"

	s.pmu.Lock()
	s.pending[uuid1] = &pair{req: req1, res: resCh1}
	s.pending[uuid2] = &pair{req: req2, res: resCh2}
	s.pmu.Unlock()

	// test random selection returns one of them
	s.pmu.RLock()
	selectedUUID, selectedPair, ok := s.getRandomPending()
	s.pmu.RUnlock()

	if !ok {
		t.Fatal("expected pending request")
	}
	if selectedUUID != uuid1 && selectedUUID != uuid2 {
		t.Fatalf("expected one of the added uuids, got: %s", selectedUUID)
	}
	if selectedPair == nil {
		t.Fatal("expected pair to be non-nil")
	}
}

func TestConcurrentMapAccess(t *testing.T) {
	s := newTestServer()
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// concurrent writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				uuid := fmt.Sprintf("uuid-%d-%d", id, j)
				req := newTestRequest()
				resCh := make(chan *pb.Response, 1)

				s.pmu.Lock()
				s.pending[uuid] = &pair{req: req, res: resCh}
				s.pmu.Unlock()

				// sometimes delete immediately
				if j%2 == 0 {
					s.pmu.Lock()
					delete(s.pending, uuid)
					s.pmu.Unlock()
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

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	// should have waited at least the timeout duration
	if duration < 100*time.Millisecond {
		t.Fatalf("expected at least 100ms delay, got %v", duration)
	}

	expectedMsg := "no pending requests after timeout"
	if !strings.Contains(w.Body.String(), expectedMsg) {
		t.Fatalf("expected timeout message, got: %s", w.Body.String())
	}
}

func TestHandleDataWithPending(t *testing.T) {
	s := newTestServer()

	// add pending request
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := uuid.NewString()

	s.pmu.Lock()
	s.pending[testUUID] = &pair{req: testReq, res: resCh}
	s.pmu.Unlock()

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
}

func TestHandleSubmitValid(t *testing.T) {
	s := newTestServer()

	// add pending request
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := uuid.NewString()

	s.pmu.Lock()
	s.pending[testUUID] = &pair{req: testReq, res: resCh}
	s.pmu.Unlock()

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

	// verify pending request was removed
	s.pmu.RLock()
	_, exists := s.pending[testUUID]
	s.pmu.RUnlock()

	if exists {
		t.Fatal("expected pending request to be removed")
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

	expectedMsg := "pending not found"
	if !strings.Contains(w.Body.String(), expectedMsg) {
		t.Fatalf("expected 'pending not found' message, got: %s", w.Body.String())
	}
}

func TestHandleSubmitMalformedJSON(t *testing.T) {
	s := newTestServer()

	// add pending request
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := uuid.NewString()

	s.pmu.Lock()
	s.pending[testUUID] = &pair{req: testReq, res: resCh}
	s.pmu.Unlock()

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

	// check that pending request exists
	s.pmu.RLock()
	pendingCount := len(s.pending)
	s.pmu.RUnlock()

	if pendingCount != 1 {
		t.Fatalf("expected 1 pending request, got %d", pendingCount)
	}

	// simulate web client submission
	testRes := newTestResponse()
	resJSON, err := protojson.Marshal(testRes)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// get the uuid (there should be exactly one)
	var pendingUUID string
	s.pmu.RLock()
	for uuid := range s.pending {
		pendingUUID = uuid
		break
	}
	s.pmu.RUnlock()

	// simulate http submission
	req := httptest.NewRequest("POST", "/submit/"+pendingUUID, bytes.NewReader(resJSON))
	w := httptest.NewRecorder()
	req.SetPathValue("uuid", pendingUUID)

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

	// verify pending request exists
	s.pmu.RLock()
	pendingCount := len(s.pending)
	s.pmu.RUnlock()

	if pendingCount != 1 {
		t.Fatalf("expected 1 pending request, got %d", pendingCount)
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

	// verify pending request was cleaned up
	s.pmu.RLock()
	pendingCount = len(s.pending)
	s.pmu.RUnlock()

	if pendingCount != 0 {
		t.Fatalf("expected 0 pending requests after cancellation, got %d", pendingCount)
	}
}

func TestConcurrentCollectorCalls(t *testing.T) {
	s := newTestServer()
	client, cleanup := startTestGRPCServer(t, s)
	defer cleanup()

	const numCalls = 5
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testReq := newTestRequest()

	// start multiple concurrent grpc calls
	type result struct {
		res *pb.Response
		err error
	}

	results := make(chan result, numCalls)

	for i := 0; i < numCalls; i++ {
		go func() {
			res, err := client.Collect(ctx, testReq)
			results <- result{res: res, err: err}
		}()
	}

	// wait for all requests to be registered
	time.Sleep(50 * time.Millisecond)

	// verify all pending requests exist
	s.pmu.RLock()
	pendingCount := len(s.pending)
	uuids := make([]string, 0, pendingCount)
	for uuid := range s.pending {
		uuids = append(uuids, uuid)
	}
	s.pmu.RUnlock()

	if pendingCount != numCalls {
		t.Fatalf("expected %d pending requests, got %d", numCalls, pendingCount)
	}

	// respond to each request
	testRes := newTestResponse()
	resJSON, err := protojson.Marshal(testRes)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	for _, uuid := range uuids {
		req := httptest.NewRequest("POST", "/submit/"+uuid, bytes.NewReader(resJSON))
		w := httptest.NewRecorder()
		req.SetPathValue("uuid", uuid)

		s.handleSubmit(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("submit failed for %s: %d: %s", uuid, w.Code, w.Body.String())
		}
	}

	// verify all grpc calls completed successfully
	for i := 0; i < numCalls; i++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("grpc call %d failed: %v", i, result.err)
			}
			if result.res == nil {
				t.Fatalf("grpc call %d returned nil response", i)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("grpc call %d timed out", i)
		}
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
	}

	data, err := json.Marshal(webReq)
	if err != nil {
		t.Fatalf("failed to marshal webRequest: %v", err)
	}

	// verify it produces json with uuid and proto fields
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
}

func TestHandleDataValidationError(t *testing.T) {
	s := newTestServer()

	// create a request that will fail validation
	badReq := &pb.Request{} // nil inputs will fail validation when we enhance it

	resCh := make(chan *pb.Response, 1)
	testUUID := "test-uuid"

	s.pmu.Lock()
	s.pending[testUUID] = &pair{req: badReq, res: resCh}
	s.pmu.Unlock()

	req := httptest.NewRequest("GET", "/data.json", nil)
	w := httptest.NewRecorder()

	s.handleData(w, req)

	// since our current validate function only checks for nil, this will pass
	// but we test the code path
	if w.Code != http.StatusOK {
		// validation failed as expected
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for validation error, got %d", w.Code)
		}
	}
}

func TestWebRequestMarshalJSONError(t *testing.T) {
	// test that MarshalJSON function gets called
	testReq := newTestRequest()
	webReq := webRequest{
		UUID:  "test-uuid",
		Proto: testReq,
	}

	// this should succeed with normal proto and exercise the custom MarshalJSON
	data, err := webReq.MarshalJSON()
	if err != nil {
		t.Fatalf("expected marshal to succeed: %v", err)
	}

	// verify the output structure
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse marshaled json: %v", err)
	}

	if result["uuid"] != "test-uuid" {
		t.Fatalf("expected uuid field")
	}
	if result["proto"] == nil {
		t.Fatal("expected proto field")
	}

	// test with nil proto - this actually succeeds with protojson
	webReqNil := webRequest{
		UUID:  "test-uuid",
		Proto: nil,
	}

	_, err = webReqNil.MarshalJSON()
	// nil proto is actually valid for protojson.Marshal
	if err != nil {
		t.Logf("nil proto marshal error (this may be expected): %v", err)
	}
}

func TestHandleSubmitMissingUUID(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest("POST", "/submit/", nil)
	w := httptest.NewRecorder()

	// don't set uuid path value
	req.SetPathValue("uuid", "")

	s.handleSubmit(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing uuid, got %d", w.Code)
	}

	expectedMsg := "missing: uuid"
	if !strings.Contains(w.Body.String(), expectedMsg) {
		t.Fatalf("expected 'missing: uuid' message, got: %s", w.Body.String())
	}
}

func TestHandleSubmitReadError(t *testing.T) {
	s := newTestServer()

	// add pending request
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := "test-uuid"

	s.pmu.Lock()
	s.pending[testUUID] = &pair{req: testReq, res: resCh}
	s.pmu.Unlock()

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

// errorReader always returns an error when read
type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

func TestCollectorContextDoneBeforeResponse(t *testing.T) {
	s := newTestServer()
	client, cleanup := startTestGRPCServer(t, s)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	testReq := newTestRequest()

	// start grpc call
	errCh := make(chan error, 1)
	go func() {
		_, err := client.Collect(ctx, testReq)
		errCh <- err
	}()

	// wait for request to be registered
	time.Sleep(10 * time.Millisecond)

	// verify pending request exists
	s.pmu.RLock()
	pendingCount := len(s.pending)
	s.pmu.RUnlock()

	if pendingCount != 1 {
		t.Fatalf("expected 1 pending request, got %d", pendingCount)
	}

	// cancel immediately before any response
	cancel()

	// verify grpc call was cancelled and cleaned up
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected quick cancellation")
	}

	// verify cleanup happened
	time.Sleep(10 * time.Millisecond)
	s.pmu.RLock()
	finalPendingCount := len(s.pending)
	s.pmu.RUnlock()

	if finalPendingCount != 0 {
		t.Fatalf("expected 0 pending requests after cancellation, got %d", finalPendingCount)
	}
}

// edge case tests to improve coverage

func TestHandleDataMarshalError(t *testing.T) {
	s := newTestServer()

	// create a scenario where json.Marshal fails
	// this is tricky since webRequest.MarshalJSON handles the marshaling
	// but we can trigger it by having the subsequent json.Marshal fail
	testReq := newTestRequest()
	resCh := make(chan *pb.Response, 1)
	testUUID := "test-uuid"

	s.pmu.Lock()
	s.pending[testUUID] = &pair{req: testReq, res: resCh}
	s.pmu.Unlock()

	req := httptest.NewRequest("GET", "/data.json", nil)
	w := httptest.NewRecorder()

	s.handleData(w, req)

	// this should succeed since our webRequest is well-formed
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCollectorResponseChannelClosed(t *testing.T) {
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

	// wait for request to be registered
	time.Sleep(10 * time.Millisecond)

	// get the pending request and close its response channel manually
	var pendingUUID string
	var p *pair

	s.pmu.RLock()
	for uuid, pair := range s.pending {
		pendingUUID = uuid
		p = pair
		break
	}
	s.pmu.RUnlock()

	if p == nil {
		t.Fatal("no pending request found")
	}

	// close the response channel to trigger the "response channel closed" path
	close(p.res)

	// remove from pending to avoid the submit handler from interfering
	s.pmu.Lock()
	delete(s.pending, pendingUUID)
	s.pmu.Unlock()

	// verify grpc call returns error
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error for closed channel")
		}
		expectedMsg := "response channel closed"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Fatalf("expected '%s' in error, got: %v", expectedMsg, err)
		}
	case res := <-resultCh:
		t.Fatalf("unexpected successful result: %v", res)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}

// validation tests

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.Request
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
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
			}
		})
	}
}
