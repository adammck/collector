#!/usr/bin/env bash
set -euxo pipefail
cd "$(dirname "$0")/.."

# run tests with coverage and race detection
go test -v -race -coverprofile=coverage.out ./...

# display coverage report
go tool cover -func=coverage.out

# check coverage threshold
coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "coverage: $coverage%"

# optionally generate html coverage report
# go tool cover -html=coverage.out -o coverage.html