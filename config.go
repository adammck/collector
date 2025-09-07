package main

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort           int
	GRPCPort           int
	MaxPendingRequests int
	HTTPTimeout        time.Duration
	SubmitTimeout      time.Duration
}

func loadConfig() *Config {
	cfg := &Config{
		HTTPPort:           8000,
		GRPCPort:           50051,
		MaxPendingRequests: 1000,
		HTTPTimeout:        30 * time.Second,
		SubmitTimeout:      5 * time.Second,
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.HTTPPort = p
		}
	}

	if port := os.Getenv("GRPC_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.GRPCPort = p
		}
	}

	if limit := os.Getenv("MAX_PENDING_REQUESTS"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			cfg.MaxPendingRequests = l
		}
	}

	if timeout := os.Getenv("HTTP_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			cfg.HTTPTimeout = t
		}
	}

	if timeout := os.Getenv("SUBMIT_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			cfg.SubmitTimeout = t
		}
	}

	return cfg
}