.PHONY: all build test bench fuzz lint sec vet fmt align check clean build-wasm build-tinygo

all: check build

build:
	go build ./...

test:
	go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

bench:
	go test -bench=. -benchmem ./...

fuzz:
	@echo "Running fuzz tests for 10s per target..."
	@for pkg in $$(go list ./...); do \
		go test -fuzz=. -fuzztime=10s $$pkg 2>/dev/null || true; \
	done

lint:
	golangci-lint run ./...

sec:
	gosec ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .
	goimports -w .

align:
	fieldalignment -fix ./...

check: fmt vet lint sec test

clean:
	rm -rf bin/ dist/ build/ coverage.out coverage.html *.prof *.pprof

build-wasm:
	GOOS=js GOARCH=wasm go build -o /dev/null ./...

build-tinygo:
	tinygo build -o /dev/null -target=wasm ./...
