GO ?= go
GORELEASER ?= goreleaser
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
REPOSITORY ?= hust-open-atom-club/atomgit-cli
NOTES_FILE ?=
RELEASE_NAME ?=
PRERELEASE ?=

.DEFAULT_GOAL := build

.PHONY: all build install uninstall test test-race vet lint fmt fmt-check coverage release release-snapshot publish clean help

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
		echo "VERSION is required (example: make release VERSION=v0.5.0)"; \
		exit 1; \
	}
	GORELEASER=$(GORELEASER) TAG=$(VERSION) ./scripts/build-release.sh

release-snapshot:
	@test -n "$(VERSION)" || { \
		echo "VERSION is required (example: make release-snapshot VERSION=v0.5.0)"; \
		exit 1; \
	}
	AG_RELEASE_SNAPSHOT=1 GORELEASER=$(GORELEASER) TAG=$(VERSION) ./scripts/build-release.sh

publish:
	@test -n "$(VERSION)" || { \
		echo "VERSION is required (example: make publish VERSION=v0.5.0 NOTES_FILE=notes.md)"; \
		exit 1; \
	}
	@test -n "$(NOTES_FILE)" || { \
		echo "NOTES_FILE is required (example: make publish VERSION=v0.5.0 NOTES_FILE=notes.md)"; \
		exit 1; \
	}
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) build
	$(MAKE) release VERSION="$(VERSION)" GORELEASER="$(GORELEASER)"
	AG_RELEASE_CLI="$(abspath $(BIN))" node scripts/publish-atomgit-release.js \
		--repo "$(REPOSITORY)" \
		--version "$(VERSION)" \
		--dir "dist/$(VERSION)" \
		--notes-file "$(NOTES_FILE)" \
		--target "$$(git rev-parse HEAD)" $(if $(RELEASE_NAME),--name "$(RELEASE_NAME)",) $(if $(filter 1 true yes,$(PRERELEASE)),--prerelease,)

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
	@echo "  make release VERSION=vX.Y.Z Build a tagged release from a clean worktree"
	@echo "  make release-snapshot VERSION=vX.Y.Z"
	@echo "                              Build local test archives without tag validation"
	@echo "  make publish VERSION=vX.Y.Z NOTES_FILE=notes.md"
	@echo "                              Validate, build, upload, and verify an AtomGit Release"
	@echo "  make clean                  Remove local build, release, and coverage files"
