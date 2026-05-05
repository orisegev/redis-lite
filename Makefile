.PHONY: build run test fmt lint clean

build:
	go build -o bin/server ./cmd/server/

run:
	go run ./cmd/server/

test:
	go test -race ./...

fmt:
	gofmt -w .

lint:
	go vet ./...

clean:
	rm -rf bin/
