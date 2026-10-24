.PHONY: run build test vet fmt docker

# Load .env if present
run:
	set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

docker:
	docker build -t 42-ad-bachelors-backend .
