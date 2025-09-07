# Collector

A flexible web application for collecting training data by presenting users with 
various types of input visualizations and collecting their responses. Supports 
multiple simultaneous visualization types for complex multi-modal scenarios.

The system uses a queue-based approach that allows humans to work through 
training data requests sequentially at their own pace, with the ability to
defer difficult or ambiguous cases for later review. This makes it suitable
for both batch collection and real-time scenarios like robotics applications.

I'm trying to use this to generate training data which is suitable for my very
simple control model in [rl-sandbox][]. I want a robot to move towards the red
box, but when I manually capture training data from the actual pixels (via my
eyeballs) using a simulator, it barely works at all, because at runtime the
pixels first pass through a vision model. This program is intended to present
me with the same input data that the control model gets, so I can provide
examples based only on that.


## Installation

```console
$ git clone ...
$ brew install protoc-gen-go protoc-gen-go-grpc
$ bin/gen-proto.sh
$ cd frontend && npm install  # install frontend dependencies
$ npm run build               # build React frontend
$ cd ..
$ bin/test.sh                 # run tests (74.2% coverage)
```


## Usage

```console
$ go run main.go
```

By default, the HTTP server runs on port 8000 and the gRPC server on port 50051.
Open http://localhost:8000 in your browser to start collecting training data.

### Development

For frontend development with hot module replacement:

```console
# Terminal 1: Start the Go server
$ go run main.go

# Terminal 2: Start the frontend dev server
$ cd frontend && npm run dev
```

The dev server runs on port 5173 with proxy to the Go backend.

### Visualization Types

The system supports multiple types of data visualization:

- **Grid**: 2D grids for spatial data (e.g., game states, occupancy maps)
- **Multi-Channel Grid**: RGB images, depth maps, or multi-sensor grid data
- **Scalar**: Single values with progress bars (temperature, speed, confidence)
- **Vector2D**: Directional data with arrow visualization (velocity, forces)
- **Time Series**: Temporal data with line charts (sensor readings over time)

Multiple visualizations can be displayed simultaneously with automatic layout management.

### Web Interface

**Modern React frontend** (migrated from vanilla JS in 2024):
- **Professional dashboard**: header, main content panels, and footer layout  
- **Dynamic visualization layout**: automatically arranges 1-N visualizations
- **Multi-modal support**: grids, scalars, vectors, and time series in one view
- **Interactive options**: gradient-styled cards with visible keyboard shortcuts
- **Live status indicators**: animated queue status and color-coded state messages
- **Modern design**: Inter font, gradients, shadows, and smooth transitions
- **TypeScript**: full type safety for all API interactions and UI components

### Keyboard Shortcuts

- **Option hotkeys**: Press the displayed key (1, 2, etc.) to select an option
- **Ctrl+D**: Defer the current item to review later
- **Ctrl+N**: Fetch the next item (same as clicking "Fetch Data")

### Queue Management

The system maintains a FIFO queue of training requests:
- Items are processed in order of arrival
- Deferred items move to the end of the queue
- Queue status is displayed in the interface
- Maximum of 1000 pending requests

### API Endpoints

- `GET /data.json` - Get next training data item
- `POST /submit/{uuid}` - Submit response for a specific item
- `POST /defer/{uuid}` - Defer an item and get the next one
- `GET /queue/status` - Get current queue statistics

### Real-time Usage

For robotics or simulation scenarios where training data is collected live:
- The queue handles continuous streams of requests
- gRPC clients should implement appropriate timeouts
- Consider implementing fallback actions for time-sensitive decisions
- The system maintains request order for temporal consistency

## Examples

The `examples/` directory contains sample gRPC clients demonstrating different visualization types:

- `examples/grid/` - Simple 2D grid visualization (original example)
- `examples/multi_channel_grid/` - RGB image data with 3-channel visualization
- `examples/scalar/` - Temperature sensor with progress bar display
- `examples/vector/` - 2D velocity vector with arrow visualization  
- `examples/time_series/` - Sensor readings over time with line chart
- `examples/multi_input/` - Complex robotics scenario with depth camera + velocity + temperature

Run any example:
```console
$ go run examples/scalar/main.go
$ go run examples/multi_input/main.go
```

## Architecture

### Backend (Go)
- **gRPC service** on port 50051 for training data requests
- **HTTP server** on port 8000 serving React frontend and JSON API
- **Thread-safe queue** with FIFO ordering and defer functionality
- **Multi-visualization support** with validation for all data types
- **Comprehensive validation** for grids, scalars, vectors, time series, and multi-channel data
- **Structured error handling** with proper HTTP status codes

### Frontend (React + TypeScript)
- **Vite** for fast development and optimized production builds
- **React Query** for server state management with built-in retry logic
- **Zustand** for lightweight client state management
- **Tailwind CSS** for utility-first styling
- **Full type safety** with TypeScript interfaces matching protobuf structures

## License

MIT


[rl-sandbox]: https://github.com/adammck/rl-sandbox
