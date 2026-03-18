BINDIR := bin
LDFLAGS := -X github.com/mel-project/mel/internal/version.Version=0.1.0-dev

.PHONY: fmt vet lint test build build-agent build-cli verify smoke

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

verify: lint test build

smoke:
	./scripts/smoke.sh
