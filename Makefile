.PHONY: test test-coverage build build-cli build-server run run-server run-cli clean

test:
	go test -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

build-cli:
	go build -o bin/icm-cli ./cmd/cli

build-server:
	go build -o bin/icm-server ./cmd/server

build: build-cli build-server

run-server:
	go run ./cmd/server

run-cli:
	go run ./cmd/cli

run: run-server

clean:
	rm -rf bin/
