.PHONY: test build build-cli build-server run run-server run-cli clean

test:
	go test ./...

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
