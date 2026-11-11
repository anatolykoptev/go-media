# go-media

**Module**: `github.com/anatolykoptev/go-media`
**Repo**: `/home/user/src/go-media/`

## Architecture

Pipeline library: Extract → Download → Audio → Transcribe.
- `Extractor` interface: platform-specific video URL extraction
- `Transcriber` interface: audio-to-text conversion
- `Processor`: orchestrates the full pipeline
- See `docs/ARCHITECTURE.md` for details; every document is indexed in [docs/INDEX.md](docs/INDEX.md)

## Commands

```bash
make test    # run tests
make lint    # golangci-lint
make build   # build all packages
```

## Rules

- Source files ≤ 200 lines
- No logging — return errors, let consumer log
- No init() magic — explicit registration
- exec.Command for ffmpeg/ffprobe (no cgo)
- ffmpeg/ffprobe are required test deps (CI installs them); absence fails loudly, never skips
- Use `GOWORK=off` for all go commands (parent go.work exists)

## Packages

| Package | Purpose |
|---------|---------|
| `media` (root) | Core types, interfaces, processor, download, audio |
| `extract/instagram` | Instagram/Threads extractor (go-threads) |
| `extract/youtube` | YouTube extractor (kkdai → go-ytdlp → ox-browser fallback) |
| `extract/twitter` | Twitter/X syndication — planned |
| `extract/reddit` | Reddit DASH — planned |
| `transcribe/gostt` | go-stt wrapper for OpenAI-compatible STT services (the separate `transcribe/openai` package was removed in #25) |
| `extract/dash` | MPEG-DASH manifest parsing and representation selection |

## Gate

CI (`.github/workflows/preflight.yml`, `ubuntu-24.04-arm`) installs ffmpeg and runs `make preflight`:

```sh
make preflight   # go build ./..., gofmt -l ., go vet ./..., go test -race -coverprofile ./..., govulncheck
```

`govulncheck` is pinned at v1.6.0 and its failure does not fail the target (`|| true`); golangci-lint
is pinned at v2.14.0 in the Makefile (`make lint`). A nightly workflow repeats the suite. A docs-only
PR (Markdown, `docs/`, licence, templates) skips the heavy job through the `docs-scope` job in
`preflight.yml`; that block is canonical across the fleet and compared byte for byte, so do not edit
it here. Versions come from release-please (Conventional Commits); never tag or edit `CHANGELOG.md`
by hand.

## Invariants that are easy to break

- **The pipeline is Extract, Download, Audio, Transcribe;** a new platform adds an `Extractor`, a
  new STT backend a `Transcriber`. Do not add platform logic to `Processor`.
- **Slides are not transcribed.** `processSlides` downloads carousel images and returns per-slide
  errors in order; whether to transcribe video slides is the consumer's call.
- **Downloads honour `Options.MaxSize`** (0 means no limit) and a cancelled download removes its
  partial file.
- **Cut boundaries are faded** (`audioFadeFilter`) to avoid clicks; keep that when changing clip code.
- **This repository is public.** No hostnames, operator paths, account names, tokens or personal
  data in any tracked file, tests and fixtures included.
