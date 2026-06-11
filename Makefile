.PHONY: run build lint fmt test docker-up tools

run:
	go run ./cmd/server/main.go

build:
	go build -o bin/redpen-checker ./cmd/server/main.go

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

test:
	go test ./... -v -count=1

docker-up:
	docker compose up -d

tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest