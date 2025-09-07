# Collector Service Development Notes

## Testing

### Running Tests
Use the single test runner script for consistent results:
```bash
./bin/test.sh
```

This runs tests with race detection and generates coverage reports. Current coverage is 67.9% of source code (excluding generated protobuf files).

### Test Structure
- `main_test.go` contains comprehensive unit and integration tests for queue-based architecture
- `queue_test.go` contains dedicated queue functionality tests
- Tests cover concurrent operations, HTTP handlers, gRPC service, end-to-end flows, and input validation
- Race condition testing with `-race` flag
- Configurable timeout for test scenarios (server.timeout field)
- Extensive validation test coverage with 95+ test cases for all edge cases
- Queue tests verify FIFO ordering, defer operations, and concurrent access

### Key Testing Patterns
- Use `newTestServer()` helper for test server instances with queue-based architecture
- Set `s.timeout` to short durations (100ms) for timeout tests
- Custom JSON unmarshaling requires parsing as `map[string]interface{}` due to protojson format
- Mock error readers with custom `Read()` method for testing error paths
- Validation tests use table-driven approach with expected error message fragments
- Test data must match grid dimensions (e.g., 10x10 grid needs 100 data values)
- **Error response testing**: expect structured JSON errors, not plain text messages
- **Status code changes**: missing UUID changed from 404→400, timeout errors changed from 404→408
- **Queue-based testing**: Tests use `s.queue.Enqueue()` and `s.current` map instead of old pending/waiter system
- **Architecture migration**: Tests updated from direct map+mutex to queue-based approach in 2024

## Architecture

### Core Components
- **server**: manages queue and current active requests with `server.queue *Queue` and `server.current map[string]*QueueItem`
- **queue.go**: thread-safe FIFO queue with defer functionality and waiter notifications
- **HTTP handlers**: `/data.json` (polling), `/submit/{uuid}` (responses), `/defer/{uuid}` (defer), `/queue/status` (statistics)
- **gRPC service**: `Collect` RPC with context cancellation support
- **concurrency**: thread-safe queue operations with RWMutex, waiter channel notifications for efficient polling
- **architecture change**: migrated from direct `pending map[string]*pair` + mutex to queue-based system

### Request Flow
1. gRPC `Collect` call validates input and enqueues request with response channel
2. HTTP `/data.json` dequeues next request (30s timeout) and serves to web client
3. Web client can either:
   - Submit response via `/submit/{uuid}` (completes the request)
   - Defer via `/defer/{uuid}` (moves item to end of queue and serves next)
4. Response flows back through gRPC channel to complete the `Collect` call
5. Context cancellation properly removes items from queue

### Input Validation
- **Request validation**: ensures at least one input is provided
- **Grid validation**: positive dimensions, max 100x100 size limit, data array matches grid size
- **Data validation**: checks for NaN/Inf values in floats, validates data types
- **Output schema validation**: requires 2+ options with unique single-character hotkeys and non-empty labels
- Validation occurs at both gRPC entry point and HTTP data serving
- Clear error messages with context about which field failed validation

### JSON Marshaling
- `webRequest` has custom `MarshalJSON()` using protojson for proto field
- Includes queue status in web responses for UI display
- Protobuf fields use capitalized names in JSON (e.g. "Visualization", "Data")
- Parse responses as `map[string]interface{}` rather than struct unmarshaling

## Build System
- `bin/gen-proto.sh` - regenerates protobuf code
- `bin/test.sh` - runs tests with coverage (includes both main and queue tests)
- protobuf files in `proto/` generate code in `proto/gen/`

## Error Handling

### gRPC Error Management
- **Error helpers** (`errors.go`): proper grpc status codes with monitoring integration
  - `validationError()` → InvalidArgument
  - `notFoundError()` → NotFound  
  - `timeoutError()` → DeadlineExceeded
  - `internalError()` → Internal
  - `resourceExhaustedError()` → ResourceExhausted
- **Resource limits**: max 1000 pending requests in queue to prevent memory exhaustion
- **Request lifecycle**: proper cleanup on all exit paths, queue items removed on context cancellation

### HTTP Error Responses
- **Structured JSON errors** (`http_errors.go`): consistent error format with code, message, and optional details
- **Error categorization**: timeouts (408), validation errors (400), not found (404), internal errors (500)
- **Client-friendly messages**: actionable error descriptions for debugging

### Client Retry Logic
- **Exponential backoff** (`client/retry.go`): configurable retry with increasing delays
- **Retryable codes**: Unavailable, ResourceExhausted, DeadlineExceeded
- **Circuit breaker pattern**: max attempts with backoff multiplier and ceiling

### JavaScript Error Handling  
- **Automatic retry**: network errors and timeouts trigger exponential backoff
- **Error categorization**: distinguishes between client errors, server errors, and network issues
- **User feedback**: clear state messages with retry indication
- **Queue integration**: defer operations gracefully handle errors and provide fallback to next item

### Monitoring
- **Error statistics** (`monitoring.go`): atomic counters for different error types
- **Error tracking**: validation, timeout, internal, and resource exhaustion metrics
- **Performance monitoring**: integrated into error helper functions

## Development Notes
- Main function starts both HTTP (port 8000) and gRPC (port 50051) servers
- Static files served from `./static/` directory
- Concurrent access tested extensively - no race conditions detected in queue operations
- Context cancellation properly cleans up queue items and pending requests
- **No panic recovery**: system fails fast rather than attempting recovery from unknown state

## Queue System

### Queue Operations
- **FIFO ordering**: items processed in arrival order (except deferred items)
- **Defer functionality**: moves items to end of queue for later processing
- **Thread safety**: all operations protected by RWMutex for concurrent access
- **Waiter notifications**: efficient polling through channel-based notifications
- **Resource limits**: maximum 1000 items in queue

### Queue Status Tracking
- **Active items**: ready for processing
- **Deferred items**: moved to end of queue, skipped during normal processing
- **Total items**: sum of active and deferred
- **Real-time updates**: status included in all web responses

### UI Integration
- **Queue display**: shows position and totals in interface
- **Keyboard shortcuts**: Ctrl+D to defer, Ctrl+N for next item
- **Defer button**: visual UI element with tooltip
- **Status updates**: real-time queue information

## Real-time Usage Considerations

### Robotics/Live Scenarios
- **FIFO ordering** maintains temporal consistency for sequential robot actions
- **Context timeouts** should be set appropriately by gRPC clients for time-sensitive operations
- **Fallback strategies** recommended for critical decisions when human response is delayed
- **Queue monitoring** helps track operator workload and response times

### Simulation Integration
- **Pausable environments** (like MuJoCo) can wait indefinitely for human responses
- **Defer functionality** useful for ambiguous cases that need additional context
- **Batch processing** supported through queue accumulation during simulation pauses

### Performance Notes
- **HTTP polling** adds latency (~1 request/response cycle per decision)
- **Queue overhead** minimal for typical workloads (< 1000 pending items)
- **Memory usage** scales linearly with queue size
- **Concurrent safety** verified through extensive race condition testing