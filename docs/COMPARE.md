# Competitive Analysis

## Video Downloaders (Go)

| Library | Stars | Platforms | Deps | Approach | Status |
|---------|-------|-----------|------|----------|--------|
| iawia002/lux | ~31k | ~50 | 69 | Registry + platform extractors | Active |
| kkdai/youtube | ~3.5k | 1 (YouTube) | 15 | Native Innertube API | Active |
| lrstanley/go-ytdlp | ~1k | 3000+ | Python | yt-dlp CLI wrapper | Active |

## STT Clients

| Library | Stars | Type | Backends | io.Reader |
|---------|-------|------|----------|-----------|
| sashabaranov/go-openai | ~9.5k | Full SDK | OpenAI/compatible | Yes |
| anatolykoptev/go-stt | internal | Focused | HTTP + WebSocket | Yes |
| mutablelogic/go-whisper | ~300 | Server | whisper.cpp + OpenAI | Yes |

## FFmpeg Wrappers

| Library | Stars | Health | Approach | Status |
|---------|-------|--------|----------|--------|
| u2takey/ffmpeg-go | ~4k | B (75) | DAG (ffmpeg-python port) | Declining |
| xfrr/goffmpeg | ~1k | C (65) | Struct-based config | Low activity |
| exec.Command | — | — | Direct subprocess | Always works |

## Rust Alternatives

| Library | Stars | Purpose | vs Go |
|---------|-------|---------|-------|
| rustube | ~1.5k | YouTube only | Fragile, YouTube-only |
| whisper-rs | ~1.5k | whisper.cpp bindings | Same C++ underneath |
| ac-ffmpeg | ~250 | FFmpeg bindings | Same C underneath |
| wreq | — | TLS fingerprint HTTP | ox-browser already covers |
| candle-whisper | ~18k | Pure Rust Whisper | 2-3x slower on ARM64 |

## Platform Video API Status (March 2026)

| Platform | Auth | Proxy | Anti-bot | Approach |
|----------|------|-------|----------|----------|
| Instagram | Cookies/App token | Residential | Very High | go-threads (GraphQL+SSR) |
| YouTube | PO Token (2025+) | Recommended | High | kkdai/youtube + PO sidecar |
| TikTok | Session cookies | Residential | Very High | ox-browser fetch-smart |
| Twitter/X | None (syndication) | Recommended | Medium | Syndication API |
| Reddit | None (public) | No | Low | JSON API + DASH merge |
| Telegram | Bot token | No | None | Bot API getFile |
| VK | OAuth token | No | Low | Video API |

## Key Architectural Patterns (from competitors)

1. **Registry pattern** (lux): `Register(name, extractor)` at init, URL dispatch by match
2. **io.Reader streaming** (kkdai): No temp files, pipe directly to next stage
3. **Auto-binary install** (go-ytdlp): Download yt-dlp binary with PGP verification
4. **DAG pipeline** (ffmpeg-go): Build processing graph, compile to ffmpeg command

## go-media Positioning

**Gap**: No library combines download + audio extract + transcribe.
**Approach**: Lightweight orchestrator with pluggable extractors and transcribers.
**Differentiators**: Pipeline API, chunked transcription, no Python dependency for core platforms.
