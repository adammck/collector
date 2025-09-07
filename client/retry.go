package client

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/adammck/collector/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	RetryableCodes    []codes.Code
}

var DefaultRetryConfig = RetryConfig{
	MaxAttempts:       3,
	InitialBackoff:    1 * time.Second,
	MaxBackoff:        30 * time.Second,
	BackoffMultiplier: 2.0,
	RetryableCodes: []codes.Code{
		codes.Unavailable,
		codes.ResourceExhausted,
		codes.DeadlineExceeded,
	},
}

func CollectWithRetry(ctx context.Context, client pb.CollectorClient, 
	req *pb.Request, cfg RetryConfig) (*pb.Response, error) {
	
	var lastErr error
	backoff := cfg.InitialBackoff
	
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			
			backoff = time.Duration(float64(backoff) * cfg.BackoffMultiplier)
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
		
		resp, err := client.Collect(ctx, req)
		if err == nil {
			return resp, nil
		}
		
		lastErr = err
		
		// check if retryable
		st, ok := status.FromError(err)
		if !ok {
			return nil, err  // not a grpc error
		}
		
		retryable := false
		for _, code := range cfg.RetryableCodes {
			if st.Code() == code {
				retryable = true
				break
			}
		}
		
		if !retryable {
			return nil, err
		}
		
		log.Printf("attempt %d failed with %v, retrying...", attempt+1, st.Code())
	}
	
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}