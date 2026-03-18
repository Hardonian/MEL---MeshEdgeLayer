BINDIR := bin
LDFLAGS := -X github.com/mel-project/mel/internal/version.Version=0.1.0-rc1

.PHONY: fmt vet lint test build build-agent build-cli build-cross verify smoke

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
