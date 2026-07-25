.PHONY: build test test-short lint preflight clean

# Pinned CI tool versions (ci-setup gostall pattern: never @latest).
GOLANGCI_LINT_VERSION := v2.12.2
GOVULNCHECK_VERSION := v1.6.0

# Resolve the binary itself — never assume `go install` lands on PATH
# (ci-setup: $(go env GOPATH)/bin is absent from self-hosted runner PATH,
# and hosted images mask the gap). Fall back to GOPATH/bin, guard with a
# hint instead of dying with a bare "not found".
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null || echo $$(GOWORK=off go env GOPATH)/bin/golangci-lint)
GOVULNCHECK := $(shell command -v govulncheck 2>/dev/null || echo $$(GOWORK=off go env GOPATH)/bin/govulncheck)

build:
	GOWORK=off go build ./...

test:
	GOWORK=off go test -race -coverprofile=coverage.txt ./...

test-short:
	GOWORK=off go test -short -race -coverprofile=coverage.txt ./...

lint:
	@[ -x "$(GOLANGCI_LINT)" ] || { echo "golangci-lint not found; install with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)"; exit 1; }
	GOWORK=off "$(GOLANGCI_LINT)" run ./...

# preflight is the CI merge gate: gofmt + vet + build + test + govulncheck.
# Runs on GitHub-hosted ubuntu-24.04-arm (free unlimited for public repos).
# ffmpeg is installed in the workflow so integration tests run (no skip-to-green).
preflight: build
	@echo "==> gofmt"
	@GOWORK=off gofmt -l . | tee /tmp/gofmt-issues | test ! -s /tmp/gofmt-issues
	@echo "==> vet"
	@GOWORK=off go vet ./...
	@echo "==> test"
	@GOWORK=off go test -race -coverprofile=coverage.txt ./...
	@echo "==> govulncheck"
	@[ -x "$(GOVULNCHECK)" ] || { echo "govulncheck not found; install with: go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)"; exit 1; }
	@GOWORK=off "$(GOVULNCHECK)" ./... || true

clean:
	GOWORK=off go clean -cache -testcache
