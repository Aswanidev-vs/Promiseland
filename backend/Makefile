.PHONY: tidy fmt vet test build run docker

tidy:
	go mod tidy

fmt:
	gofmt -w $(shell find . -name '*.go' -type f)

vet:
	go vet ./...

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd/server

docker:
	docker build -t promisedland-backend:local .
