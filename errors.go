package main

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validation errors -> InvalidArgument
func validationError(msg string, args ...any) error {
	recordError(codes.InvalidArgument)
	return status.Errorf(codes.InvalidArgument, msg, args...)
}

// not found errors -> NotFound
func notFoundError(resource string, id string) error {
	recordError(codes.NotFound)
	return status.Errorf(codes.NotFound, "%s not found: %s", resource, id)
}

// timeout errors -> DeadlineExceeded
func timeoutError(operation string) error {
	recordError(codes.DeadlineExceeded)
	return status.Errorf(codes.DeadlineExceeded, "%s timed out", operation)
}

// server errors -> Internal
func internalError(err error) error {
	recordError(codes.Internal)
	return status.Errorf(codes.Internal, "internal error: %v", err)
}

// resource exhaustion -> ResourceExhausted
func resourceExhaustedError(resource string) error {
	recordError(codes.ResourceExhausted)
	return status.Errorf(codes.ResourceExhausted, "%s limit exceeded", resource)
}