#!/usr/bin/env bash
set -euxo pipefail
cd "$(dirname "$0")/.."

# run tests with coverage and race detection
go test -v -race -coverprofile=coverage.out ./...

# display coverage report (exclude generated protobuf files)
go tool cover -func=coverage.out | grep -v "proto/gen/"

# check coverage threshold (main package only, excluding generated files and examples)  
coverage=$(go test -v -race -coverprofile=coverage.out ./... 2>/dev/null | grep "github.com/adammck/collector.*coverage:" | head -1 | awk '{print $5}' | sed 's/%//')
echo "coverage: $coverage%"

# optionally generate html coverage report
# go tool cover -html=coverage.out -o coverage.html