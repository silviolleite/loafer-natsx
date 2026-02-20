SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

GO              ?= go
GOIMPORTS        = goimports
GOLANGCI_LINT    = golangci-lint
GOFIELDALIGNMENT = fieldalignment
GOVULNCHECK		 = govulncheck
GOLANGCI_VERSION = v2.8.0
GOPATH_BIN       = $(shell $(GO) env GOPATH)/bin
LOCAL_PREFIX     = github.com/silviolleite/loafer-natsx

COVER_TMP        = cover.out.tmp
COVER_OUT        = cover.out
TEST_COVER_TMP   = tmp.out
TEST_COVER_OUT   = geral.out

.PHONY: help clean format lint configure install-golang-ci install-goimports install-fieldalignment install-govulncheck cover cover-html test test-chaos update-dependencies setup-dev

help:
	@echo "Targets:"
	@echo "  clean                Clean test cache"
	@echo "  format               Format code with goimports"
	@echo "  lint                 Format and run golangci-lint"
	@echo "  configure            Install tools (Go + git hooks)"
	@echo "  cover                Coverage report (cover.out)"
	@echo "  cover-html           Coverage HTML report (cover.html)"
	@echo "  test-chaos           Stress test"
	@echo "  test                 Tests with race + coverage (geral.out)"
	@echo "  update-dependencies  Update dependencies and run go mod tidy"
	@echo "  setup-dev            Install npm (if needed) + husky + commitlint"

clean:
	@$(GO) clean -testcache

check-vuln:
	@$(GOVULNCHECK) ./...

format:
	@$(GOIMPORTS) -local $(LOCAL_PREFIX) -w -l .
	@$(GOFIELDALIGNMENT) -fix ./...

lint: format
	@$(GOLANGCI_LINT) run --allow-parallel-runners ./... --max-same-issues 0

install-golang-ci:
	@echo "Installing golangci-lint $(GOLANGCI_VERSION)"
	@curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOPATH_BIN) $(GOLANGCI_VERSION)
	@echo "golangci-lint installed successfully"

install-goimports:
	@echo "Installing goimports"
	@$(GO) install golang.org/x/tools/cmd/goimports@latest
	@echo "goimports installed successfully"

install-fieldalignment:
	@echo "Installing fieldalignment"
	@$(GO) install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	@echo "fieldalignment installed successfully"

install-govulncheck:
	@echo "Installing govulncheck"
	@$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "govulncheck installed successfully"

setup-dev:
	@echo "==> Checking npm..."
	@if ! command -v npm &> /dev/null; then \
		echo "npm not found. Installing Node.js via nvm..."; \
		curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash; \
		export NVM_DIR="$$HOME/.nvm"; \
		[ -s "$$NVM_DIR/nvm.sh" ] && . "$$NVM_DIR/nvm.sh"; \
		nvm install --lts; \
	else \
		echo "npm found: $$(npm --version)"; \
	fi
	@echo "==> Installing dependencies (husky + commitlint)..."
	@npm install
	@echo "==> Configuring commit-msg hook..."
	@echo 'npx --no -- commitlint --edit "$$1"' > .husky/commit-msg
	@chmod +x .husky/commit-msg
	@echo "==> Setup complete! Git hooks enabled."

configure: install-golang-ci install-goimports install-fieldalignment install-govulncheck setup-dev

cover:
	@$(GO) test -covermode=count -coverprofile=$(COVER_TMP) ./...
	@grep -v fake $(COVER_TMP) > $(COVER_OUT)
	@$(GO) tool cover -func=$(COVER_OUT)

cover-html: cover
	@$(GO) tool cover -html=$(COVER_OUT) -o cover.html
	@echo "Generated cover.html"

test-chaos: clean
	@GOMAXPROCS=1 $(GO) test ./... -race -count=30 -shuffle=on -timeout 15m

test: clean check-vuln
	@$(GO) test -timeout 1m -race -covermode=atomic -coverprofile=$(TEST_COVER_TMP) -coverpkg=./... ./...
	@grep -Ev 'examples' $(TEST_COVER_TMP) > $(TEST_COVER_OUT)
	@$(GO) tool cover -func=$(TEST_COVER_OUT)

update-dependencies:
	@$(GO) get -t -u ./... && $(GO) mod tidy
