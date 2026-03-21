BINDIR := bin
VERSION := 0.1.0-rc1
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "now")
LDFLAGS := -X github.com/mel-project/mel/internal/version.Version=$(VERSION) \
	-X github.com/mel-project/mel/internal/version.GitCommit=$(COMMIT) \
	-X github.com/mel-project/mel/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: fmt vet lint test build build-agent build-cli build-cross verify smoke version

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

lint: fmt vet

test:
	go test ./...

build: build-agent build-cli

build-agent:
	mkdir -p $(BINDIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-agent ./cmd/mel-agent

build-cli:
	mkdir -p $(BINDIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel ./cmd/mel

build-cross:
	mkdir -p $(BINDIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-linux-amd64 ./cmd/mel
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-linux-arm64 ./cmd/mel

verify: lint test build build-cross

smoke:
	./scripts/smoke.sh

version:
	@echo "MEL Version Information:"
	@echo "  Version:           $(VERSION)"
	@echo "  Git Commit:        $(COMMIT)"
	@echo "  Build Time:        $(BUILD_TIME)"
	@echo "  Schema Version:    15"
	@echo "  Compatibility:     dev"
	@go run -ldflags "$(LDFLAGS)" ./cmd/mel version
