# Collector Service Development Notes

## Testing

### Running Tests
Use the single test runner script for consistent results:
```bash
./bin/test.sh
```

This runs tests with race detection and generates coverage reports. Current coverage is 71.8% of source code (excluding generated protobuf files).

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
- Global `config` variable must be initialized in `TestMain()` for tests to work
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

### Code Organization (2024 Refactoring)
- **main.go**: server startup, configuration loading, graceful shutdown (120 lines)
- **handlers.go**: HTTP endpoint handlers and web request logic (188 lines)
- **grpc.go**: gRPC service implementation with structured logging (67 lines)
- **validation.go**: comprehensive input validation functions (354 lines)
- **config.go**: environment-based configuration management (57 lines)
- **queue.go**: thread-safe FIFO queue with defer functionality and waiter notifications
- **errors.go**: gRPC error helpers with monitoring integration
- **http_errors.go**: structured HTTP error responses
- **monitoring.go**: error statistics and metrics collection

### Core Components
- **server**: manages queue and current active requests with `server.queue *Queue` and `server.current map[string]*QueueItem`
- **HTTP handlers**: `/data.json` (polling), `/submit/{uuid}` (responses), `/defer/{uuid}` (defer), `/queue/status` (statistics), `/metrics` (monitoring), `/health` (health checks)
- **gRPC service**: `Collect` RPC with context cancellation support and multi-visualization support
- **concurrency**: thread-safe queue operations with RWMutex, optimized waiter notifications (wake only one waiter)
- **observability**: structured logging (slog), metrics endpoint, health checks
- **configuration**: environment variables with command-line fallbacks
- **graceful shutdown**: SIGTERM/SIGINT handling with 30s timeout
- **visualization system**: supports Grid, MultiChannelGrid, Scalar, Vector2D, and TimeSeries types with comprehensive validation

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
- **Visualization validation**: comprehensive validation for all visualization types
  - **Grid**: positive dimensions, max 100x100 size, data array matches grid size
  - **MultiChannelGrid**: channel count validation (max 10), optional channel names
  - **Scalar**: label required, min < max, single float value within range
  - **Vector2D**: label required, positive max_magnitude, exactly 2 float values
  - **TimeSeries**: label required, positive points (max 1000), min < max, all values in range
- **Data validation**: checks for NaN/Inf values in floats, validates data types
- **Output schema validation**: requires 2+ options with unique single-character hotkeys and non-empty labels
- Validation occurs at both gRPC entry point and HTTP data serving
- Clear error messages with context about which field failed validation

### JSON Marshaling
- `webRequest` has custom `MarshalJSON()` using protojson for proto field
- Includes queue status in web responses for UI display
- Protobuf fields use capitalized names in JSON (e.g. "Visualization", "Data")
- Parse responses as `map[string]interface{}` rather than struct unmarshaling

## Configuration Management

### Environment Variables
- `HTTP_PORT` - HTTP server port (default: 8000)
- `GRPC_PORT` - gRPC server port (default: 50051)
- `MAX_PENDING_REQUESTS` - queue size limit (default: 1000)
- `HTTP_TIMEOUT` - timeout for HTTP data polling (default: 30s)
- `SUBMIT_TIMEOUT` - timeout for response submission (default: 5s)

### Command Line Flags
Still supported for backwards compatibility:
- `-http-port` - HTTP server port
- `-grpc-port` - gRPC server port

Command line flags override environment variables when both are present.

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

### Observability & Monitoring
- **Structured logging** (`slog`): replaced printf debugging with structured logs
- **Metrics endpoint** (`/metrics`): queue statistics, error counts, and request totals
- **Health endpoint** (`/health`): service status with timestamp and queue info
- **Error statistics** (`monitoring.go`): atomic counters for different error types  
- **Error tracking**: validation, timeout, internal, and resource exhaustion metrics
- **Performance monitoring**: integrated into error helper functions
- **Graceful shutdown**: proper server lifecycle management with 30s timeout

## Development Notes
- **Modular startup**: main function loads config, starts servers, handles signals
- **Static files served** from `./frontend/dist/` directory (React build output)  
- **Environment configuration**: supports both env vars and command-line flags
- **Graceful shutdown**: servers stop cleanly on SIGTERM/SIGINT with 30s timeout
- **Concurrent access tested** extensively - no race conditions detected in queue operations
- **Context cancellation** properly cleans up queue items and pending requests  
- **Optimized queue**: notifyWaiters() wakes only one waiter to reduce thundering herd
- **Structured logging**: informative log messages with context fields
- **No panic recovery**: system fails fast rather than attempting recovery from unknown state

### Pre-commit Checklist
**ALWAYS** before committing changes:
1. **Check IDE diagnostics** - resolve all compilation errors and warnings
2. **Run tests** - `./bin/test.sh` must pass with no failures
3. **Build frontend** - `cd frontend && npm run build` must succeed
4. **Verify coverage** - ensure test coverage remains reasonable (target 65%+)
5. **Architecture alignment** - ensure tests match current queue-based architecture (not old pending/waiter system)

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

## Frontend Architecture

### Modern React Stack (2024)
Migrated from vanilla JavaScript (~450 loc) to modern React + TypeScript:
- **Vite** - fast dev server with HMR, optimized production builds
- **React 18** - component-based UI with hooks for state management
- **TypeScript** - full type safety for protobuf structures and API contracts
- **Tailwind CSS** - utility-first styling, replacing manual CSS
- **React Query** - server state management with built-in retry/caching/error handling
- **Zustand** - lightweight client state (current UUID, app state, queue status)

### Build System
- **Development**: `cd frontend && npm run dev` (with Vite dev server and API proxy)
- **Production**: `cd frontend && npm run build` → outputs to `frontend/dist/`
- **Server integration**: Go server serves static files from `./frontend/dist/`
- **Proxy configuration**: Vite dev server proxies API calls to Go backend on port 8000

### Component Architecture
- **App.tsx** - React Query provider, retry configuration
- **CollectorApp.tsx** - main application logic, data fetching orchestration, handles multiple inputs
- **VisualizationLayout.tsx** - dynamic layout manager for 1-N visualizations (grid, side-by-side, 2x2, etc.)
- **VisualizationRenderer.tsx** - router component that selects appropriate visualization based on type
- **GridVisualization.tsx** - renders 2D grid data with table-based styling
- **MultiChannelGridVisualization.tsx** - renders RGB/multi-channel data on HTML5 canvas
- **ScalarVisualization.tsx** - progress bar visualization for single values with labels/units
- **Vector2DVisualization.tsx** - arrow visualization on coordinate system with magnitude display
- **TimeSeriesVisualization.tsx** - line chart with grid lines and statistical summaries
- **OptionList.tsx** - interactive option cards with hover effects and visible hotkeys
- **QueueStatus.tsx** - real-time queue statistics with live indicator
- **StateNotifier.tsx** - application state messages with icons and color coding
- **useKeyboardShortcuts** - custom hook for Ctrl+D/Ctrl+N shortcuts

### Visual Design (2024)
- **Professional UI**: dashboard layout with header, main content panels, and footer
- **Modern Styling**: gradients, shadows, rounded corners, and proper spacing
- **Typography**: Inter font integration for improved readability
- **Interactive Elements**: hover effects, transitions, and proper disabled states
- **Color System**: semantic color coding for states (green=ready, amber=waiting, red=error)
- **Responsive Layout**: card-based panels with proper visual hierarchy

### Type Safety
- **types.ts** - comprehensive TypeScript interfaces matching protobuf structures
- **API layer** - typed request/response with structured error handling
- **Component props** - fully typed for compile-time error catching

### Error Handling & Retry Logic
- **React Query retry** - exponential backoff for 408/503/network errors
- **Error categorization** - timeout vs client vs server errors with appropriate messaging
- **State management** - proper error state display and recovery flows

### Performance Optimizations
- **Bundle size** - 229kb raw → 71kb gzipped (reasonable for functionality)
- **Code splitting** - Vite handles automatic optimization
- **React optimizations** - useEffect dependencies, proper re-render patterns

### Development Notes
- **Hot module replacement** - instant updates during development
- **TypeScript compilation** - catches errors at build time, not runtime
- **Maintainable** - clear separation of concerns vs mixed vanilla JS classes
- **Extensible** - easy to add new visualization types or output formats