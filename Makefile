BINDIR := bin
# Prefer the VM-installed toolchain when PATH `go` is too old to satisfy go.mod.
GO := $(if $(wildcard /usr/local/go/bin/go),/usr/local/go/bin/go,go)
VERSION := 0.1.0-rc1
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "now")
LDFLAGS := -X github.com/mel-project/mel/internal/version.Version=$(VERSION) \
	-X github.com/mel-project/mel/internal/version.GitCommit=$(COMMIT) \
	-X github.com/mel-project/mel/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: fmt vet lint test build build-agent build-cli build-cross verify smoke version demo-verify frontend-build frontend-lint reality-check product-verify

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*' -not -path './frontend/node_modules/*')

vet:
	$(GO) vet ./...

frontend-lint:
	cd frontend && node ./scripts/require-node24.mjs && npm ci && npm run lint

# gofmt is intentionally not part of `lint` so routine lint does not rewrite the whole tree.
# Run `make fmt` before committing, or use `make verify` (which runs fmt then lint).
lint: vet frontend-lint

test:
	$(GO) test ./...

build: frontend-build build-agent build-cli

frontend-build:
	cd frontend && node ./scripts/require-node24.mjs && npm ci && npm run build
	mkdir -p internal/web/assets
	cp -r frontend/dist/* internal/web/assets/

build-agent:
	mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-agent ./cmd/mel-agent

build-cli: frontend-build
	mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel ./cmd/mel

build-cross:
	mkdir -p $(BINDIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-linux-amd64 ./cmd/mel
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-linux-arm64 ./cmd/mel

verify: fmt lint test build build-cross reality-check product-verify

smoke:
	./scripts/smoke.sh

demo-verify: build-cli
	$(GO) test ./internal/demo/...
	./bin/mel demo scenarios >/dev/null
	./scripts/demo-evidence.sh healthy-private-mesh .tmp/demo-verify.json

version:
	@echo "MEL Version Information:"
	@echo "  Version:           $(VERSION)"
	@echo "  Git Commit:        $(COMMIT)"
	@echo "  Build Time:        $(BUILD_TIME)"
	@echo "  Schema Version:    35"
	@echo "  Compatibility:     dev"
	@$(GO) run -ldflags "$(LDFLAGS)" ./cmd/mel version

reality-check:
	./scripts/repo-os-reality-check.sh


product-verify:
	./scripts/verify-product-system.sh
