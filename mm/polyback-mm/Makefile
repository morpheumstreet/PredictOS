.PHONY: all build test tidy run-executor run-strategy

all: build

tidy:
	go mod tidy

build:
	mkdir -p bin
	go build -o bin/executor ./cmd/executor
	go build -o bin/strategy ./cmd/strategy
	go build -o bin/ingestor ./cmd/ingestor
	go build -o bin/analytics ./cmd/analytics
	go build -o bin/infrastructure ./cmd/infrastructure
	go build -o bin/intelligence ./cmd/intelligence

test:
	go test ./...

run-executor: build
	bash ./scripts/run-executor.sh

run-strategy: build
	bash ./scripts/run-strategy.sh
