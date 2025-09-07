package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "github.com/adammck/collector/proto/gen"
	"google.golang.org/grpc"
)

var (
	config *Config
)

var errTimeout = errors.New("no pending requests after timeout")


type server struct {
	queue   *Queue
	current map[string]*QueueItem
	cmu     sync.RWMutex

	timeout time.Duration
}

func newServer(cfg *Config) *server {
	return &server{
		queue:   NewQueue(),
		current: make(map[string]*QueueItem),
		timeout: cfg.HTTPTimeout,
	}
}








func main() {
	// support command line flags for backwards compatibility
	hp := flag.Int("http-port", 8000, "port for http server to listen on")
	gp := flag.Int("grpc-port", 50051, "port for grpc server to listen on")
	flag.Parse()

	// load config from environment variables
	config = loadConfig()
	
	// override with command line flags if provided
	if *hp != 8000 {
		config.HTTPPort = *hp
	}
	if *gp != 50051 {
		config.GRPCPort = *gp
	}

	s := newServer(config)

	// Create HTTP server
	httpAddr := fmt.Sprintf(":%d", config.HTTPPort)
	httpSrv := &http.Server{
		Addr:    httpAddr,
		Handler: s.ServeHTTP(),
	}

	// Create gRPC server
	grpcAddr := fmt.Sprintf(":%d", config.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	pb.RegisterCollectorServer(grpcSrv, &collectorServer{s: s})

	// Start servers
	go func() {
		log.Printf("HTTP server listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	go func() {
		log.Printf("gRPC server listening on %s", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("shutting down servers...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server gracefully
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Shutdown gRPC server gracefully
	grpcSrv.GracefulStop()

	log.Println("servers stopped")
}









