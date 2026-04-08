BINDIR := bin
# Prefer the VM-installed toolchain when PATH `go` is too old to satisfy go.mod.
GO := $(if $(wildcard /usr/local/go/bin/go),/usr/local/go/bin/go,go)
VERSION := 0.1.0-rc1
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "now")
LDFLAGS := -X github.com/mel-project/mel/internal/version.Version=$(VERSION) \
	-X github.com/mel-project/mel/internal/version.GitCommit=$(COMMIT) \
	-X github.com/mel-project/mel/internal/version.BuildTime=$(BUILD_TIME)

# Single npm ci per Node workspace in this chain: frontend-verify installs once; build reuses node_modules.
RELEASE_VERIFY_TARGETS := product-verify frontend-verify site-verify test build-cli-release smoke

.PHONY: fmt vet lint test check build build-agent build-cli build-cli-release mel-cli-go build-cross verify verify-stack smoke version demo-verify demo-seed first-proof \
	frontend-node-contract frontend-install frontend-build frontend-build-reuse-deps frontend-lint frontend-typecheck frontend-test frontend-verify \
	frontend-lint-fast frontend-typecheck-fast frontend-test-fast frontend-verify-fast \
	site-verify \
	reality-check product-verify release-verify-chain check-frontend-install-churn premerge-verify premerge-verify-fast

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*' -not -path './frontend/node_modules/*')

vet:
	$(GO) vet ./...

frontend-node-contract:
	./scripts/require-node24.sh --context "make frontend-*"

frontend-install: frontend-node-contract
	rm -rf frontend/node_modules
	cd frontend && npm ci

frontend-lint: frontend-install
	cd frontend && npm run lint

frontend-typecheck: frontend-install
	cd frontend && npm run typecheck

frontend-test: frontend-install
	cd frontend && npm run test

frontend-verify: frontend-lint frontend-typecheck frontend-test

# Fast local-only frontend verification: skips clean install and expects deps already present.
# Verification chain (truthful scope):
#   - make lint          → go vet + frontend lint + public site lint (each npm project: own npm ci)
#   - make test          → Go tests only
#   - make frontend-verify / frontend-verify-fast → ESLint + tsc + vitest (install vs no-install)
#   - make site-verify / site-verify-fast → public Next.js site: ESLint + tsc + production build (single site npm ci via site-install)
#   - make build         → frontend build + copies dist → internal/web/assets/ + Go mel binary
#   - make mel-cli-go    → Go binary only; uses committed embedded assets (no npm)
#   - make smoke         → scripts/smoke.sh (needs ./bin/mel from build-cli/build)
#   - make premerge-verify → scripts/verify-release-local.sh (release-grade: chained verification incl. frontend + site npm ci)
#   - make premerge-verify-fast → same script with VERIFY_SKIP_CLEAN_INSTALL=1 (iteration only; not release-grade)
site-node-contract:
	./scripts/require-node24.sh --context "make site-*"

site-install: site-node-contract
	rm -rf site/node_modules
	cd site && npm ci

site-lint: site-install
	cd site && npm run lint

site-typecheck: site-install
	cd site && npm run typecheck

site-build: site-install
	cd site && npm run build

site-verify: site-lint site-typecheck site-build

site-lint-fast: site-node-contract
	cd site && npm run lint

site-typecheck-fast: site-node-contract
	cd site && npm run typecheck

site-build-fast: site-node-contract
	cd site && npm run build

site-verify-fast: site-lint-fast site-typecheck-fast site-build-fast

frontend-lint-fast: frontend-node-contract
	cd frontend && npm run lint

frontend-typecheck-fast: frontend-node-contract
	cd frontend && npm run typecheck

frontend-test-fast: frontend-node-contract
	cd frontend && npm run test

frontend-verify-fast: frontend-lint-fast frontend-typecheck-fast frontend-test-fast

# gofmt is intentionally not part of `lint` so routine lint does not rewrite the whole tree.
# Run `make fmt` before committing, or use `make verify` (which runs fmt then lint).
lint: vet frontend-lint site-lint

# Go packages only. Frontend tests: `make frontend-test` or `make verify-stack`.
test:
	$(GO) test ./...

# Same as `make verify-stack` — one obvious full-product check (npm ci + lint + Go test + build + smoke).
check: verify-stack

build: frontend-build build-agent build-cli

frontend-build: frontend-install
	cd frontend && npm run build
	mkdir -p internal/web/assets
	cp -r frontend/dist/* internal/web/assets/

# Reuse existing frontend/node_modules (no npm ci). Used by build-cli-release after frontend-verify in release-verify-chain.
frontend-build-reuse-deps: frontend-node-contract
	cd frontend && npm run build
	mkdir -p internal/web/assets
	cp -r frontend/dist/* internal/web/assets/

build-agent:
	mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-agent ./cmd/mel-agent

build-cli: frontend-build
	mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel ./cmd/mel

# Same binary as build-cli but skips frontend npm ci when node_modules already populated (release-verify-chain).
build-cli-release: frontend-build-reuse-deps
	mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel ./cmd/mel

# Go-only mel binary using committed embedded web assets (no npm / Node gate).
mel-cli-go:
	mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel ./cmd/mel

build-cross:
	mkdir -p $(BINDIR)
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-linux-amd64 ./cmd/mel
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINDIR)/mel-linux-arm64 ./cmd/mel

verify: fmt lint test build build-cross reality-check product-verify

# Canonical stack check: Go + frontend lint/typecheck/tests + embedded UI build + smoke.
# Use when you need one obvious “full product surface” signal (includes npm ci via lint/build).
verify-stack: lint test build smoke

smoke:
	./scripts/smoke.sh

demo-verify: build-cli
	$(GO) test ./internal/demo/...
	./bin/mel demo scenarios >/dev/null
	./scripts/demo-evidence.sh healthy-private-mesh .tmp/demo-verify.json

# One-command seeded UI dataset for contributors (fixture-backed, not live mesh proof).
DEMO_SEED_SCENARIO ?= healthy-private-mesh
DEMO_SEED_CONFIG ?= demo_sandbox/mel.demo.json

demo-seed: mel-cli-go
	@mkdir -p "$(dir $(DEMO_SEED_CONFIG))"
	./bin/mel demo init-sandbox --out "$(DEMO_SEED_CONFIG)" >/dev/null
	chmod 600 "$(DEMO_SEED_CONFIG)"
	./bin/mel demo seed --scenario "$(DEMO_SEED_SCENARIO)" --config "$(DEMO_SEED_CONFIG)"
	@echo "Seeded scenario $(DEMO_SEED_SCENARIO) into $(DEMO_SEED_CONFIG). Run: ./bin/mel serve --config $(DEMO_SEED_CONFIG)"

# One-command first proof flow: deterministic seed + evidence bundle + explicit claim boundaries.
first-proof:
	./scripts/first-proof.sh

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

release-verify-chain:
	$(MAKE) $(RELEASE_VERIFY_TARGETS)

check-frontend-install-churn:
	./scripts/check-frontend-install-churn.sh

premerge-verify:
	./scripts/verify-release-local.sh

premerge-verify-fast:
	VERIFY_SKIP_CLEAN_INSTALL=1 ./scripts/verify-release-local.sh
