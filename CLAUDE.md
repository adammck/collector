# Collector Service Development Notes

## Testing

### Running Tests
Use the single test runner script for consistent results:
```bash
./bin/test.sh
```

This runs tests with race detection and generates coverage reports. Current coverage is 79.9% of the main package.

### Test Structure
- `main_test.go` contains comprehensive unit and integration tests
- Tests cover concurrent operations, HTTP handlers, gRPC service, end-to-end flows, and input validation
- Race condition testing with `-race` flag
- Configurable timeout for test scenarios (server.timeout field)
- Extensive validation test coverage with 95+ test cases for all edge cases

### Key Testing Patterns
- Use `newTestServer()` helper for test server instances
- Set `s.timeout` to short durations (100ms) for timeout tests
- Custom JSON unmarshaling requires parsing as `map[string]interface{}` due to protojson format
- Mock error readers with custom `Read()` method for testing error paths
- Validation tests use table-driven approach with expected error message fragments
- Test data must match grid dimensions (e.g., 10x10 grid needs 100 data values)
- **Error response testing**: expect structured JSON errors, not plain text messages
- **Status code changes**: missing UUID changed from 404→400, timeout errors changed from 404→408

## Architecture

### Core Components
- **server**: manages pending requests and waiter notifications
- **HTTP handlers**: `/data.json` (polling) and `/submit/{uuid}` (responses)  
- **gRPC service**: `Collect` RPC with context cancellation support
- **concurrency**: thread-safe pending map with RWMutex, waiter channel notifications

### Request Flow
1. gRPC `Collect` call validates input and creates pending request with response channel
2. HTTP `/data.json` polls for pending requests (30s timeout) and validates before serving
3. Web client submits response via `/submit/{uuid}`
4. Response flows back through gRPC channel

### Input Validation
- **Request validation**: ensures at least one input is provided
- **Grid validation**: positive dimensions, max 100x100 size limit, data array matches grid size
- **Data validation**: checks for NaN/Inf values in floats, validates data types
- **Output schema validation**: requires 2+ options with unique single-character hotkeys and non-empty labels
- Validation occurs at both gRPC entry point and HTTP data serving
- Clear error messages with context about which field failed validation

### JSON Marshaling
- `webRequest` has custom `MarshalJSON()` using protojson for proto field
- Protobuf fields use capitalized names in JSON (e.g. "Visualization", "Data")
- Parse responses as `map[string]interface{}` rather than struct unmarshaling

## Build System
- `bin/gen-proto.sh` - regenerates protobuf code
- `bin/test.sh` - runs tests with coverage
- protobuf files in `proto/` generate code in `proto/gen/`

## Error Handling

### gRPC Error Management
- **Error helpers** (`errors.go`): proper grpc status codes with monitoring integration
  - `validationError()` → InvalidArgument
  - `notFoundError()` → NotFound  
  - `timeoutError()` → DeadlineExceeded
  - `internalError()` → Internal
  - `resourceExhaustedError()` → ResourceExhausted
- **Resource limits**: max 1000 pending requests to prevent memory exhaustion
- **Request lifecycle**: proper cleanup on all exit paths with defer statements

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

### Monitoring
- **Error statistics** (`monitoring.go`): atomic counters for different error types
- **Error tracking**: validation, timeout, internal, and resource exhaustion metrics
- **Performance monitoring**: integrated into error helper functions

## Development Notes
- Main function starts both HTTP (port 8000) and gRPC (port 50051) servers
- Static files served from `./static/` directory
- Concurrent access tested extensively - no race conditions detected
- Context cancellation properly cleans up pending requests
- **No panic recovery**: system fails fast rather than attempting recovery from unknown state