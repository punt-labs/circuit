VERSION := $(or $(shell git describe --tags --always 2>/dev/null | sed 's/^v//'),dev)
LDFLAGS := -X main.version=$(VERSION)
MARKDOWNLINT ?= markdownlint-cli2
NPM ?= npm
PROBCLI ?= probcli
ZSPEC ?= z-spec

RUNTIME_SPEC_MACHINES ?= 2
RUNTIME_SPEC_SESSIONS ?= 2
RUNTIME_SPEC_CHECKS ?= 2
RUNTIME_SPEC_MAX_INITIALISATIONS ?= 15

GOLANGCI_LINT_VERSION := v2.12.2
GOBIN := $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif
GOLANGCI_LINT := $(GOBIN)/golangci-lint

.PHONY: help check check-engine check-go-quality check-go-all check-pi-extension check-docs check-machines check-rpc check-specs check-runtime-spec model-check-runtime-spec smoke-pi smoke-drive lint lint-engine lint-pi-extension docs test test-engine test-pi-extension test-rpc typecheck-pi-extension format-check-pi-extension format build build-engine install clean tools coverage

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-24s %s\n", $$1, $$2}'

check: format check-engine check-rpc check-pi-extension check-docs ## Run all quality gates

check-engine: lint-engine test-engine ## Validate Go engine

# Structural quality gate: wired to codeQualityPassed in tdd-flow.
# These linters are calibrated to produce a small actionable failing set.
# Tighten thresholds incrementally via the quality-ratchet pattern.
GO_QUALITY_LINTERS := dupl,gocognit,funlen,goconst,gocritic,maintidx,exhaustive,errcheck,prealloc,nonamedreturns,cyclop

check-go-quality: ## Structural quality gate (codeQualityPassed)
	$(GOLANGCI_LINT) run --enable-only=$(GO_QUALITY_LINTERS) --tests=false ./...

# Breadth report: all candidate linters, informational only (does not gate commits).
# Run this to see the full picture before choosing the next ratchet dimension.
GO_ALL_LINTERS := $(GO_QUALITY_LINTERS),errcheck,errorlint,nilnil,nilerr,wrapcheck,unparam,wastedassign,exhaustive,nestif,cyclop,nonamedreturns,prealloc,forcetypeassert,gosec,paralleltest,thelper,revive,misspell,godot,nakedret

check-go-all: ## Full Go quality breadth report (informational, not a gate)
	$(GOLANGCI_LINT) run --enable-only=$(GO_ALL_LINTERS) --tests=false ./... || true

check-rpc: test-rpc ## Validate RPC protocol logic

check-specs: check-runtime-spec model-check-runtime-spec ## Validate formal design specs

check-runtime-spec: ## Type-check the Circuit runtime Z spec with z-spec
	$(ZSPEC) check docs/spec/circuit-runtime.tex

model-check-runtime-spec: ## Model-check the Circuit runtime Z spec with ProB
	$(PROBCLI) docs/spec/circuit-runtime.tex -model_check -card MACH $(RUNTIME_SPEC_MACHINES) -card SESSION $(RUNTIME_SPEC_SESSIONS) -card CHECK $(RUNTIME_SPEC_CHECKS) -p MAX_INITIALISATIONS $(RUNTIME_SPEC_MAX_INITIALISATIONS)

check-machines: ## Validate B machines with ProB
	$(PROBCLI) machines/build-job.mch -init
	$(PROBCLI) machines/build-job.mch -model_check -nodead
	$(PROBCLI) machines/pr-watch.mch -init
	$(PROBCLI) machines/pr-watch.mch -model_check -nodead
	$(PROBCLI) machines/review-flow.mch -init
	$(PROBCLI) machines/review-flow.mch -model_check -nodead
	$(PROBCLI) machines/retry-flow.mch -init
	$(PROBCLI) machines/retry-flow.mch -model_check -nodead
	$(PROBCLI) machines/tdd-flow.mch -init
	$(PROBCLI) machines/tdd-flow.mch -model_check -nodead

lint: lint-engine ## Lint Go engine

lint-engine: ## Lint Go with golangci-lint
	@test -x $(GOLANGCI_LINT) || { echo "golangci-lint not found at $(GOLANGCI_LINT) — run 'make tools' to install $(GOLANGCI_LINT_VERSION)"; exit 1; }
	$(GOLANGCI_LINT) run ./...

test: test-engine test-rpc test-pi-extension ## Run all tests

test-engine: ## Run Go engine tests with race detection and coverage
	go test -race -count=1 -coverprofile=coverage-engine.out ./cmd/circuit ./internal/circuitb ./internal/circuitrun

test-rpc: ## Run RPC protocol tests with coverage
	go test -race -count=1 -coverprofile=coverage-rpc.out ./internal/circuitrpc

test-pi-extension: ## Run pi extension tests with coverage
	$(NPM) --prefix .pi run test

build: build-engine ## Build binary

build-engine: ## Build circuit binary
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o circuit ./cmd/circuit/

format: ## Format Go and TypeScript code
	$(GOLANGCI_LINT) fmt
	cd .pi && npx prettier --write --ignore-path .prettierignore "**/*.ts" "**/*.json" "**/*.mjs"

install: build ## Build and install to ~/.local/bin
	mkdir -p $(HOME)/.local/bin
	rm -f $(HOME)/.local/bin/circuit
	cp circuit $(HOME)/.local/bin/circuit

check-pi-extension: typecheck-pi-extension lint-pi-extension test-pi-extension ## Validate pi extension

typecheck-pi-extension: ## Typecheck pi extension
	$(NPM) --prefix .pi run typecheck

lint-pi-extension: ## Lint pi extension
	$(NPM) --prefix .pi run lint

format-check-pi-extension: ## Check pi extension formatting
	$(NPM) --prefix .pi run format:check

docs: check-docs ## Lint markdown

check-docs: ## Lint markdown
	$(MARKDOWNLINT) "**/*.md" "#node_modules" "#.pi/node_modules" "#.tmp" "#.direnv"

smoke-pi: ## Run pi RPC smoke test (requires pi + model API key)
	python3 tests/smoke/pi_rpc_smoke.py

smoke-drive: ## Run circuit-drives-pi smoke test (requires pi + model API key)
	python3 tests/smoke/circuit_drive_smoke.py

tools: ## Install development tools
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

coverage: ## Show coverage summary for all tiers
	@echo "=== Go engine ==="
	go test -coverprofile=coverage-engine.out ./cmd/circuit ./internal/circuitb ./internal/circuitrun
	go tool cover -func=coverage-engine.out | tail -1
	@echo ""
	@echo "=== Go RPC protocol ==="
	go test -coverprofile=coverage-rpc.out ./internal/circuitrpc
	go tool cover -func=coverage-rpc.out | tail -1
	@echo ""
	@echo "=== TypeScript pi extension ==="
	$(NPM) --prefix .pi run test

clean: ## Remove build artifacts
	rm -f circuit circuit-rpc-spike coverage.out coverage-engine.out coverage-rpc.out docs/spec/*.fuzz
	rm -rf dist
