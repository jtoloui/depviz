VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: tidy fmt vet test coverage lint build clean

all: tidy fmt vet test lint build

tidy:
	go mod tidy

fmt: tidy
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -race ./...

coverage: vet
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "---"
	@go tool cover -func=coverage.out | grep total:

lint: test
	golangci-lint run ./...

build: lint
	go build $(LDFLAGS) -o bin/depviz .

clean:
	rm -rf bin/ coverage.out coverage.html
