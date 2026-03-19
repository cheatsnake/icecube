.PHONY: test test-coverage build-cli build-server run run-server clean docker-build docker-rebuild docker-up-dev docker-up-prod docker-down

test:
	go test -cover ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

build-cli:
	go build -o bin/cli ./cmd/cli

build-server:
	go build -o bin/server ./cmd/server

run-server:
	go run ./cmd/server

run: run-server

clean:
	rm -rf bin/

docker-build:
	docker build -t icecube .

docker-up-dev:
	docker compose --profile dev up -d

docker-up-prod:
	docker compose --profile prod up -d

docker-down:
	docker compose --profile dev down
	docker compose --profile prod down
