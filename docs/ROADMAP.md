# go-media Roadmap

## v0.1.0 — Core + Instagram (MVP)

Extract Instagram/Threads code from vaelor into go-media.

- [ ] Core types: `Media`, `Quality`, `Transcription`, `Chunk`, `Result`
- [ ] `Extractor` interface + `Registry`
- [ ] `Transcriber` interface
- [ ] `Processor` orchestrator (extract → download → audio → transcribe)
- [ ] HTTP downloader with timeout + max size
- [ ] FFmpeg audio extraction + duration probe + chunking
- [ ] `extract/instagram` — port from vaelor (uses go-threads)
- [ ] `transcribe/openai` — OpenAI-compatible HTTP backend
- [ ] `transcribe/gostt` — go-stt wrapper
- [ ] Tests: unit + mock-based
- [ ] Update vaelor to import `go-media` instead of internal code
- [ ] golangci-lint v2 + pre-commit + Makefile

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
- [ ] CI/CD with GitHub Actions
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
