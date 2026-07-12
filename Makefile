GO ?= go
BINARY := ag
COMMAND := ./cmd/ag
BIN_DIR := bin

EXE :=
ifeq ($(shell $(GO) env GOOS),windows)
EXE := .exe
endif

BIN := $(BIN_DIR)/$(BINARY)$(EXE)
COVERAGE_FILE ?= coverage.out
VERSION ?=

.DEFAULT_GOAL := build

.PHONY: all build install uninstall test test-race vet lint fmt fmt-check coverage release clean help

all: lint test build

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -o $(BIN) $(COMMAND)

install:
	$(GO) install $(COMMAND)

uninstall:
	@rm -f "$$($(GO) env GOPATH)/bin/$(BINARY)$(EXE)"

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint: fmt-check vet

fmt:
	$(GO) fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || { \
		echo "The following files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	}

coverage:
	$(GO) test ./... -coverprofile=$(COVERAGE_FILE)
	$(GO) tool cover -func=$(COVERAGE_FILE)

release:
	@test -n "$(VERSION)" || { \
		echo "VERSION is required (example: make release VERSION=v1.0)"; \
		exit 1; \
	}
	TAG=$(VERSION) ./scripts/build-release.sh

clean:
	@rm -rf $(BIN_DIR) dist
	@rm -f $(COVERAGE_FILE)

help:
	@echo "AtomGit CLI development targets:"
	@echo ""
	@echo "Build and installation:"
	@echo "  make build                  Build a local binary at $(BIN)"
	@echo "  make install                Build and install to GOPATH/bin"
	@echo "  make uninstall              Remove the binary from GOPATH/bin"
	@echo ""
	@echo "Checks:"
	@echo "  make test                   Run the standard test suite"
	@echo "  make test-race              Run the full test suite with race detection"
	@echo "  make lint                   Check formatting and run go vet (no file changes)"
	@echo "  make coverage               Run tests and generate $(COVERAGE_FILE)"
	@echo ""
	@echo "Maintenance:"
	@echo "  make fmt                    Format Go source files in place"
	@echo "  make release VERSION=vX.Y.Z Build cross-platform release archives"
	@echo "  make clean                  Remove local build, release, and coverage files"
