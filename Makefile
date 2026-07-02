# Zero build/test/lint targets. AGENTS.MD says "Build with `make`" and "Run `make
# lint` before opening a PR" — these targets back those instructions.
BENCH_TIME ?= 3s
PKG ?= ./internal/...

.DEFAULT_GOAL := build
.PHONY: build build-all test test-race vet fmt fmt-check lint tidy clean help bench bench-all bench-redaction

# Build the main CLI binary into ./zero.
build:
	go build -o zero ./cmd/zero

# Build every command in cmd/.
build-all:
	go build ./...

# Run the full test suite with the race detector (matches CI expectations).
test:
	go test ./... -race -count=1

# Faster, no race detector.
test-quick:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $(shell git ls-files '*.go')

# Fail if any tracked Go file is not gofmt-clean.
fmt-check:
	@out="$$(gofmt -l $$(git ls-files '*.go'))"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

# Lint = formatting check + vet (no extra tooling required).
lint: fmt-check vet

tidy:
	go mod tidy

bench:
	go test -p=1 -bench=. -benchmem -benchtime=$(BENCH_TIME) $(PKG)

bench-all:
	go test -p=1 -bench=. -benchmem -benchtime=$(BENCH_TIME) ./...

bench-redaction:
	go test -p=1 -bench=. -benchmem -benchtime=$(BENCH_TIME) ./internal/redaction/...

clean:
	rm -f zero
	go clean ./...

help:
	@echo "Targets: build (default), build-all, test, test-quick, vet, fmt, fmt-check, lint, tidy, clean"
