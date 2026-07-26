# go-media

**Module**: `github.com/anatolykoptev/go-media`
**Repo**: `/home/user/src/go-media/`

## Architecture

Pipeline library: Extract → Download → Audio → Transcribe.
- `Extractor` interface: platform-specific video URL extraction
- `Transcriber` interface: audio-to-text conversion
- `Processor`: orchestrates the full pipeline
- See `docs/ARCHITECTURE.md` for details

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
| `transcribe/openai` | OpenAI-compatible STT (ox-whisper, Groq) |
| `transcribe/gostt` | go-stt wrapper |
