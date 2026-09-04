GO ?= go
GORELEASER ?= goreleaser
GOVULNCHECK_VERSION := v1.7.0
GOVULNDB := https://vuln.go.dev
GO_MIN_VERSION := $(shell sed -n 's/^go[[:space:]][[:space:]]*//p' go.mod)
GO_MIN_TOOLCHAIN := go$(GO_MIN_VERSION)
GO_TOOLCHAIN := $(shell sed -n 's/^toolchain[[:space:]][[:space:]]*//p' go.mod)
BINARY := ag
COMMAND := ./cmd/ag
BIN_DIR := bin
RACE_PACKAGES ?= ./...
RACE_OPTIONS ?= atexit_sleep_ms=0
RELEASE_TARGETS ?= linux/amd64 linux/arm64 linux/loong64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
PLATFORM_TEST_TARGETS ?= darwin/amd64 windows/amd64

EXE :=
ifeq ($(strip $(GO_MIN_VERSION)),)
$(error go.mod must declare the minimum supported Go version)
endif
ifeq ($(strip $(GO_TOOLCHAIN)),)
$(error go.mod must declare the preferred release toolchain)
endif

export GOTOOLCHAIN := $(GO_TOOLCHAIN)

LOCAL_GO = GOTOOLCHAIN=local $(GO)
HOST_GOOS := $(shell $(LOCAL_GO) env GOOS 2>/dev/null)

ifeq ($(HOST_GOOS),windows)
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

.PHONY: all go-min-version go-version build cross-build install uninstall test test-min-go test-race test-platform-compile vet lint vulncheck fmt fmt-check coverage release release-snapshot publish clean help

all: lint test build

go-min-version:
	@actual="$$(GOTOOLCHAIN=$(GO_MIN_TOOLCHAIN) $(GO) env GOVERSION)"; \
	if [ "$$actual" != "$(GO_MIN_TOOLCHAIN)" ]; then \
		echo "Expected minimum Go toolchain $(GO_MIN_TOOLCHAIN), got $$actual" >&2; \
		exit 1; \
	fi; \
	echo "Minimum Go toolchain: $$actual"

go-version:
	@actual="$$($(GO) env GOVERSION)"; \
	if [ "$$actual" != "$(GO_TOOLCHAIN)" ]; then \
		echo "Expected release Go toolchain $(GO_TOOLCHAIN), got $$actual" >&2; \
		exit 1; \
	fi; \
	echo "Release Go toolchain: $$actual"

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -o $(BIN) $(COMMAND)

cross-build:
	@set -eu; \
	output_dir="$$(mktemp -d)"; \
	trap 'rm -rf "$$output_dir"' 0 HUP INT TERM; \
	for target in $(RELEASE_TARGETS); do \
		goos=$${target%/*}; \
		goarch=$${target#*/}; \
		echo "Building $$goos/$$goarch"; \
		GOOS="$$goos" GOARCH="$$goarch" CGO_ENABLED=0 \
			$(GO) build -trimpath -o "$$output_dir/ag-$$goos-$$goarch" $(COMMAND); \
	done

install:
	$(GO) install $(COMMAND)

uninstall:
	@set -eu; \
	if ! goos="$$($(LOCAL_GO) env GOOS)"; then \
		echo "Unable to query GOOS from the local Go command" >&2; \
		exit 1; \
	fi; \
	if [ -z "$$goos" ]; then \
		echo "Refusing to uninstall with an empty GOOS" >&2; \
		exit 1; \
	fi; \
	if ! gobin="$$($(LOCAL_GO) env GOBIN)"; then \
		echo "Unable to query GOBIN from the local Go command" >&2; \
		exit 1; \
	fi; \
	if [ -z "$$gobin" ]; then \
		if ! gopath="$$($(LOCAL_GO) env GOPATH)"; then \
			echo "Unable to query GOPATH from the local Go command" >&2; \
			exit 1; \
		fi; \
		case "$$goos" in \
			windows) install_root=$${gopath%%;*} ;; \
			*) install_root=$${gopath%%:*} ;; \
		esac; \
		if [ -z "$$install_root" ] || [ "$$install_root" = "/" ]; then \
			echo "Refusing to uninstall with an empty or root GOPATH" >&2; \
			exit 1; \
		fi; \
		gobin="$$install_root/bin"; \
	fi; \
	if [ -z "$$gobin" ] || [ "$$gobin" = "/" ]; then \
		echo "Refusing to uninstall with an empty or root GOBIN" >&2; \
		exit 1; \
	fi; \
	suffix=""; \
	if [ "$$goos" = "windows" ]; then suffix=".exe"; fi; \
	target="$$gobin/$(BINARY)$$suffix"; \
	echo "Removing $$target"; \
	rm -f -- "$$target"

test:
	$(GO) test ./...

test-min-go:
	GOTOOLCHAIN=$(GO_MIN_TOOLCHAIN) $(GO) test ./...

test-race:
	GORACE="$(RACE_OPTIONS)" $(GO) test -race -count=1 $(RACE_PACKAGES)

test-platform-compile:
	@set -eu; \
	for target in $(PLATFORM_TEST_TARGETS); do \
		goos=$${target%/*}; \
		goarch=$${target#*/}; \
		echo "Compiling tests for $$goos/$$goarch"; \
		GOOS="$$goos" GOARCH="$$goarch" CGO_ENABLED=0 \
			$(GO) test -exec=true ./...; \
	done

vet:
	$(GO) vet ./...

lint: fmt-check vet

vulncheck:
	@set -eu; \
	binary="$$(mktemp)"; \
	trap 'rm -f "$$binary"' 0 HUP INT TERM; \
	CGO_ENABLED=0 $(GO) build -trimpath -o "$$binary" $(COMMAND); \
	version_output="$$($(GO) version "$$binary")"; \
	echo "$$version_output"; \
	case "$$version_output" in \
		*": $(GO_TOOLCHAIN)") ;; \
		*) \
			echo "Expected vulnerability target built with $(GO_TOOLCHAIN)" >&2; \
			exit 1; \
			;; \
	esac; \
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) \
		-db="$(GOVULNDB)" -mode=binary "$$binary"

fmt:
	$(GO) fmt ./...

fmt-check:
	@gofmt_bin="$$($(GO) env GOROOT)/bin/gofmt"; \
	files="$$("$$gofmt_bin" -l .)"; \
	test -z "$$files" || { \
		echo "The following files need formatting:"; \
		printf '%s\n' "$$files"; \
		exit 1; \
	}

coverage:
	$(GO) test ./... -coverprofile=$(COVERAGE_FILE)
	$(GO) tool cover -func=$(COVERAGE_FILE)

release:
	@test -n "$(VERSION)" || { \
		echo "VERSION is required (example: make release VERSION=vX.Y.Z)"; \
		exit 1; \
	}
	GORELEASER=$(GORELEASER) TAG=$(VERSION) ./scripts/build-release.sh

release-snapshot:
	@test -n "$(VERSION)" || { \
		echo "VERSION is required (example: make release-snapshot VERSION=vX.Y.Z)"; \
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
	@echo "  make install                Build and install to GOBIN or GOPATH/bin"
	@echo "  make uninstall              Remove the binary from GOBIN or GOPATH/bin"
	@echo ""
	@echo "Checks:"
	@echo "  make go-min-version         Download and verify minimum Go $(GO_MIN_VERSION)"
	@echo "  make go-version             Download and verify release $(GO_TOOLCHAIN)"
	@echo "  make test                   Run the standard test suite"
	@echo "  make test-min-go            Run tests with minimum Go $(GO_MIN_VERSION)"
	@echo "  make test-race              Run race detection for $(RACE_PACKAGES)"
	@echo "  make test-platform-compile  Compile tests for macOS and Windows targets"
	@echo "  make lint                   Check formatting and run go vet (no file changes)"
	@echo "  make cross-build            Compile all seven supported release targets"
	@echo "  make vulncheck              Build and scan the $(GO_TOOLCHAIN) release binary"
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
