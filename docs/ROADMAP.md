# go-media Roadmap

## v0.1.0 — Core + Instagram (MVP) ✅

Extract Instagram/Threads code from vaelor into go-media.

- [x] Core types: `Media`, `Quality`, `Transcription`, `Chunk`, `Result`
- [x] `Extractor` interface + `Registry`
- [x] `Transcriber` interface
- [x] `Processor` orchestrator (extract → download → audio → transcribe)
- [x] HTTP downloader with timeout + max size
- [x] FFmpeg audio extraction + duration probe + chunking
- [x] `extract/instagram` — port from vaelor (uses go-threads)
- [x] `transcribe/openai` — OpenAI-compatible HTTP backend
- [x] `transcribe/gostt` — go-stt wrapper
- [x] Tests: 25 tests across 6 test files (root, instagram, openai)
- [x] Update vaelor to import `go-media` instead of internal code
- [x] golangci-lint v2 + pre-commit + Makefile
- [x] CI/CD: GitHub Actions (lint + test)
- [x] Documentation: design doc, architecture, compare, README

### Known Issues (v0.1.0)

- `transcribe/gostt` has no tests (needs go-stt mock or interface)
- `ChunkAndTranscribe()` silently skips failed chunks — no error count returned
- `go.sum` needs `go mod tidy` after clean clone (transitive deps from go-stealth)
- Pre-commit hook requires `GOWORK=off` (parent go.work interference)
- No `Process()` integration test (only `Extract()` tested in processor)
- Typo: `hashMultipler` → should be `hashMultiplier`

## v0.1.1 — Quality Fixes

- [ ] Add tests for `transcribe/gostt` package
- [ ] Add `Process()` integration test with mock transcriber
- [ ] Return chunk error count from `ChunkAndTranscribe()` (or log callback)
- [ ] Fix `hashMultipler` typo → `hashMultiplier`
- [ ] Add input validation to `Options` (ChunkSec > 0, MaxSize >= 0)
- [ ] Clean up partial file on download context cancellation
- [ ] Add codecov to CI workflow

## v0.2.0 — YouTube

- [ ] `extract/youtube` — kkdai/youtube integration
- [ ] Format selection (quality, audio-only)
- [ ] DASH audio+video merge (ffmpeg mux)
- [ ] YouTube-specific tests

## v0.3.0 — Twitter/X + Reddit

- [ ] `extract/twitter` — Syndication API (no auth, HLS → MP4)
- [ ] `extract/reddit` — DASH video + separate audio merge
- [ ] HLS stream download support

## v0.4.0 — Telegram + VK

- [ ] `extract/telegram` — Bot API getFile + Local Server support
- [ ] `extract/vk` — VK Video API (OAuth token)
- [ ] File size limit handling per platform

## v0.5.0 — TikTok + Advanced Features

- [ ] `extract/tiktok` — via ox-browser fetch-smart (CF bypass)
- [ ] Parallel chunk transcription
- [ ] Progress callbacks (download + transcribe)
- [ ] Retry with backoff for failed downloads

## v1.0.0 — Production Release

- [ ] API stability guarantee
- [ ] Comprehensive documentation
- [ ] GoReleaser
- [ ] Performance benchmarks
- [ ] All platform extractors battle-tested

## Future Ideas

- Subtitle generation (SRT/VTT from transcription chunks)
- Audio-only download mode (no video)
- Thumbnail extraction
- Batch URL processing
- Webhook/callback for async processing
- MCP server wrapper (expose as MCP tools)
