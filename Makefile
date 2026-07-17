.PHONY: build test test-short lint preflight clean

build:
	GOWORK=off go build ./...

test:
	GOWORK=off go test -race -coverprofile=coverage.txt ./...

test-short:
	GOWORK=off go test -short -race -coverprofile=coverage.txt ./...

lint:
	GOWORK=off golangci-lint run ./...

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
	@GOWORK=off govulncheck ./... || true

clean:
	go clean -cache -testcache
