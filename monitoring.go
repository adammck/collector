package main

import (
	"google.golang.org/grpc/codes"
	"sync/atomic"
)

type ErrorStats struct {
	ValidationErrors  int64
	TimeoutErrors     int64
	InternalErrors    int64
	ResourceExhausted int64
	TotalRequests     int64
}

var stats = &ErrorStats{}

func recordError(code codes.Code) {
	atomic.AddInt64(&stats.TotalRequests, 1)

	switch code {
	case codes.InvalidArgument:
		atomic.AddInt64(&stats.ValidationErrors, 1)
	case codes.DeadlineExceeded:
		atomic.AddInt64(&stats.TimeoutErrors, 1)
	case codes.Internal:
		atomic.AddInt64(&stats.InternalErrors, 1)
	case codes.ResourceExhausted:
		atomic.AddInt64(&stats.ResourceExhausted, 1)
	}
}

func getStats() ErrorStats {
	return ErrorStats{
		ValidationErrors:  atomic.LoadInt64(&stats.ValidationErrors),
		TimeoutErrors:     atomic.LoadInt64(&stats.TimeoutErrors),
		InternalErrors:    atomic.LoadInt64(&stats.InternalErrors),
		ResourceExhausted: atomic.LoadInt64(&stats.ResourceExhausted),
		TotalRequests:     atomic.LoadInt64(&stats.TotalRequests),
	}
}
