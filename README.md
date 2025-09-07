# Collector

This is a small web app to collect training data by presenting the user some
input data, and waiting for them to provide the output. It only supports a
single type of each, right now, but I have vague plans to make it flexible.

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
$ bin/test.sh  # run tests (67.9% coverage)
```


## Usage

```console
$ go run main.go
```

By default, the HTTP server runs on port 8000 and the gRPC server on port 50051.
Open http://localhost:8000 in your browser to start collecting training data.

### Web Interface

The web interface displays:
- Input visualization (currently supports 2D grids)
- Response options with keyboard shortcuts
- Queue status showing active, deferred, and total items

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


## License

MIT


[rl-sandbox]: https://github.com/adammck/rl-sandbox
