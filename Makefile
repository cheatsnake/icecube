.PHONY: test test-coverage build-cli build-server run-server clean docker-build docker-up-dev docker-up-prod docker-down dockerhub-build

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

clean:
	rm -rf bin/

docker-build:
	docker build -t icecube .

IMAGE_NAME ?= cheatsnake/icecube
TAG ?= latest

dockerhub-build:
	docker buildx create --use --name multiarch 2>/dev/null || docker buildx use multiarch
	docker buildx inspect --bootstrap
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE_NAME):$(TAG) --push .

docker-up-dev:
	docker compose --profile dev up -d

docker-up-prod:
	docker compose --profile prod up -d

docker-down:
	docker compose --profile dev down
	docker compose --profile prod down
