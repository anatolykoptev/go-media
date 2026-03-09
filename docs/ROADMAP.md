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

## v0.1.1 — Quality Fixes ✅

- [x] Add tests for `transcribe/gostt` package (4 tests)
- [x] Add `Process()` integration tests (3 tests: success, download error, no transcriber)
- [x] Add `FailedChunks` tracking to `ChunkAndTranscribe()`
- [x] Fix `hashMultipler` typo → `hashMultiplier`
- [x] Clean up partial file on download context cancellation
- [x] Consolidate constants into `defaults.go`
- [x] Fix pre-commit hook for go.work (GOWORK=off)
- [x] Add codecov to CI workflow
- [x] Update roadmap

Total: 29 tests, 0 lint issues.

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
