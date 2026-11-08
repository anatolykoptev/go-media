# Contributing to go-media

Changes are welcome when they keep the exported API stable and are covered by tests.

## Development

Requires Go 1.26, `ffmpeg` and `ffprobe` on PATH (the tests need them and fail loudly without them),
and golangci-lint v2.14.0 for linting.

```sh
make build        # GOWORK=off go build ./...
make test         # GOWORK=off go test -race -coverprofile=coverage.txt ./...
make test-short   # the same with -short
make lint         # golangci-lint run ./...
make preflight    # build, gofmt, vet, test, govulncheck
```

CI (`.github/workflows/preflight.yml`) installs ffmpeg and runs `make preflight`; a nightly workflow
repeats it. A pull request that changes only documents skips the heavy job.

House rules (also in [CLAUDE.md](CLAUDE.md)): source files stay under 200 lines, the library does
not log (it returns errors), there is no `init()` registration, and ffmpeg is driven through
`exec.Command`, never cgo.

## Commits

Conventional Commits (`feat:`, `fix:`, `perf:`, `refactor:`, `docs:`, `deps:`, `chore:`, `ci:`,
`test:`). release-please reads them to cut versions and write [CHANGELOG.md](CHANGELOG.md); do not
tag or edit the changelog by hand.

## Pull requests

- Branch from `main`, keep the change focused, and fill in the pull-request template.
- Add tests for behaviour changes; do not skip a test because ffmpeg is missing.
- Update [docs/INDEX.md](docs/INDEX.md) when you add or rename a document.
- Do not add secrets, hostnames or personal data to any file; the repository is public.

## Reporting problems

Bugs and feature requests go through the issue templates. Report vulnerabilities privately as
described in [SECURITY.md](SECURITY.md).
