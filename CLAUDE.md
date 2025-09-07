# Collector Service Development Notes

## Testing

### Running Tests
Use the single test runner script for consistent results:
```bash
./bin/test.sh
```

This runs tests with race detection and generates coverage reports. Current coverage is 76% of the main package.

### Test Structure
- `main_test.go` contains comprehensive unit and integration tests
- Tests cover concurrent operations, HTTP handlers, gRPC service, and end-to-end flows
- Race condition testing with `-race` flag
- Configurable timeout for test scenarios (server.timeout field)

### Key Testing Patterns
- Use `newTestServer()` helper for test server instances
- Set `s.timeout` to short durations (100ms) for timeout tests
- Custom JSON unmarshaling requires parsing as `map[string]interface{}` due to protojson format
- Mock error readers with custom `Read()` method for testing error paths

## Architecture

### Core Components
- **server**: manages pending requests and waiter notifications
- **HTTP handlers**: `/data.json` (polling) and `/submit/{uuid}` (responses)  
- **gRPC service**: `Collect` RPC with context cancellation support
- **concurrency**: thread-safe pending map with RWMutex, waiter channel notifications

### Request Flow
1. gRPC `Collect` call creates pending request with response channel
2. HTTP `/data.json` polls for pending requests (30s timeout)
3. Web client submits response via `/submit/{uuid}`
4. Response flows back through gRPC channel

### JSON Marshaling
- `webRequest` has custom `MarshalJSON()` using protojson for proto field
- Protobuf fields use capitalized names in JSON (e.g. "Visualization", "Data")
- Parse responses as `map[string]interface{}` rather than struct unmarshaling

## Build System
- `bin/gen-proto.sh` - regenerates protobuf code
- `bin/test.sh` - runs tests with coverage
- protobuf files in `proto/` generate code in `proto/gen/`

## Development Notes
- Main function starts both HTTP (port 8000) and gRPC (port 50051) servers
- Static files served from `./static/` directory
- Concurrent access tested extensively - no race conditions detected
- Context cancellation properly cleans up pending requests