.PHONY: build install test clean

# Build variables
BINARY_NAME=driftdetector
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse HEAD)
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS=-ldflags "-X 'main.Version=$(VERSION)' \
                  -X 'main.Commit=$(COMMIT)' \
                  -X 'main.Date=$(BUILD_DATE)' \
                  -X 'main.GoVersion=$(GO_VERSION)'"

# Build the application
build:
	go build -o bin/$(BINARY_NAME) $(LDFLAGS) ./cmd/driftdetector

# Install the application
install:
	go install $(LDFLAGS) ./cmd/driftdetector

# Run tests
test:
	go test -v -coverprofile=coverage.out ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out

# Run the application
dev: build
	./bin/$(BINARY_NAME) --help

# Run with race detector
race:
	go run -race $(LDFLAGS) ./cmd/driftdetector --help

# Build for multiple platforms
release:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 $(LDFLAGS) ./cmd/driftdetector
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 $(LDFLAGS) ./cmd/driftdetector
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe $(LDFLAGS) ./cmd/driftdetector
