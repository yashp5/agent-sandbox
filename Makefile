SHELL := /bin/sh
.DEFAULT_GOAL := all

GO ?= go
BUF ?= buf
CMD ?= sandbox-control
ARGS ?=

.PHONY: all proto generate proto-lint proto-format check-buf build run test vet clean help

# Keep generation ahead of compilation, including with make -j.
all: proto
	$(MAKE) build

check-buf:
	@command -v "$(BUF)" >/dev/null 2>&1 || { \
		echo "Buf is required. Install it with 'brew install bufbuild/buf/buf' or set BUF=/path/to/buf." >&2; \
		exit 1; \
	}

# Buf discovers every .proto in the api/proto module, including nested packages.
proto: proto-lint
	$(BUF) generate

generate: proto

proto-lint: check-buf
	$(BUF) lint
	$(BUF) format --diff --exit-code

proto-format: check-buf
	$(BUF) format -w

# Compile every package; write any implemented command executables to bin/.
# Generated Go files are checked in, so ordinary builds do not need the BSR.
build:
	@set -eu; \
	commands="$$($(GO) list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./...)"; \
	if test -n "$$commands"; then \
		mkdir -p bin; \
		$(GO) build -o bin/ ./...; \
	else \
		$(GO) build ./...; \
		echo "Packages compiled; no command binaries are implemented yet."; \
	fi

run:
	@if ! test -d "cmd/$(CMD)" || ! test -n "$$(find "cmd/$(CMD)" -maxdepth 1 -name '*.go' -print)"; then \
		echo "No Go command exists at cmd/$(CMD) yet. Use make run CMD=<command> once its main package is implemented." >&2; \
		exit 1; \
	fi
	$(GO) run ./cmd/$(CMD) $(ARGS)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

# Generated Go files are source artifacts; keep them available for offline builds.
clean:
	rm -rf ./bin

help:
	@printf '%s\n' \
		'Usage: make [target]' \
		'' \
		'  all           Generate protobufs, then build (default)' \
		'  proto         Lint schemas and generate Go/gRPC code into gen/' \
		'  generate      Alias for proto' \
		'  proto-lint    Check protobuf lint and formatting' \
		'  proto-format  Format protobuf files in place' \
		'  build         Compile all packages and write command binaries to bin/' \
		'  run           Run cmd/$(CMD); override CMD and ARGS as needed' \
		'  test          Run tests in all packages' \
		'  vet           Run go vet in all packages' \
		'  clean         Remove compiled binaries; preserve generated Go sources'
