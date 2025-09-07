package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/adammck/collector/proto/gen"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type collectorServer struct {
	pb.UnsafeCollectorServer
	s *server
}

func (cs *collectorServer) Collect(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	slog.Info("collect request received",
		"input_count", len(req.Inputs),
		"has_output_schema", req.Output != nil)

	// validate first
	if err := validate(req); err != nil {
		return nil, validationError("invalid request: %v", err)
	}

	// check resource limits
	queueStatus := cs.s.queue.Status()
	if queueStatus.Total >= config.MaxPendingRequests {
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