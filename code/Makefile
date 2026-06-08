.PHONY: build build-dev build-cross release install-dev run test clean

BINARY=bin/tick

# Build metadata injected via -ldflags.
VERSION         ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDTIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -X 'main.Version=$(VERSION)' \
          -X 'main.Commit=$(COMMIT)' \
          -X 'main.BuildTime=$(BUILDTIME)'

# Build binary to bin/tick.
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/tick

# Build with source path marker for self-update.
build-dev:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -X 'main.SourcePath=$(PWD)'" -o $(BINARY) ./cmd/tick
	@echo $(PWD) > bin/.tick-source

# Cross-platform build: 5 OS/ARCH combos.
build-cross:
	@mkdir -p bin
	@for pair in darwin:amd64 darwin:arm64 linux:amd64 linux:arm64 windows:amd64; do \
		OS=$${pair%%:*}; ARCH=$${pair##*:}; \
		EXT=""; [ "$$OS" = "windows" ] && EXT=".exe"; \
		OUT="bin/tick-$(VERSION)-$$OS-$$ARCH$$EXT"; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH \
			go build -ldflags "$(LDFLAGS)" \
			-o $$OUT ./cmd/tick; \
		sha256sum $$OUT | awk '{print $$1}' > $$OUT.sha256; \
		echo "built $$OUT"; \
	done
	@cd bin && sha256sum tick-$(VERSION)-* > SHA256SUMS

# Release build: outputs to bin/release/.
release:
	@mkdir -p bin/release
	@for pair in darwin:amd64 darwin:arm64 linux:amd64 linux:arm64 windows:amd64; do \
		OS=$${pair%%:*}; ARCH=$${pair##*:}; \
		EXT=""; [ "$$OS" = "windows" ] && EXT=".exe"; \
		OUT="bin/release/tick-$(VERSION)-$$OS-$$ARCH$$EXT"; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH \
			go build -ldflags "$(LDFLAGS)" \
			-o $$OUT ./cmd/tick; \
		sha256sum $$OUT | awk '{print $$1}' > $$OUT.sha256; \
		echo "built $$OUT"; \
	done
	@cd bin/release && sha256sum tick-$(VERSION)-* > SHA256SUMS

# Install to /usr/local/bin.
install-dev: build-dev
	install -m 0755 $(BINARY) /usr/local/bin/tick

run:
	go run ./cmd/tick serve

test:
	go test ./...

clean:
	rm -rf bin
