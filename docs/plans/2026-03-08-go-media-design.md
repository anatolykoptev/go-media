# go-media Design Document

**Date**: 2026-03-08
**Module**: `github.com/anatolykoptev/go-media`
**Status**: Approved

## Problem

No Go library combines video downloading from social platforms + audio extraction + speech transcription. Vaelor has a working but tightly-coupled implementation for Instagram/Threads (400 LOC). Need a universal, reusable library.

## Market Analysis

### Video Downloaders (Go)
- **iawia002/lux** (~31k stars): 50+ platforms, registry pattern, CLI-first. Heavy (69 deps).
- **kkdai/youtube** (~3.5k stars): YouTube-only, io.Reader streaming, lightweight (15 deps).
- **lrstanley/go-ytdlp** (~1k stars): yt-dlp wrapper, 3000+ sites, requires Python runtime.

### STT Clients
- **sashabaranov/go-openai** (~9.5k stars): OpenAI-compatible, io.Reader support, lightweight.
- **go-stt** (internal): Already used in vaelor, HTTP + WebSocket.

### FFmpeg Wrappers
- **u2takey/ffmpeg-go** (~4k stars): DAG model, declining activity.
- Direct `exec.Command` — simplest, zero deps, sufficient for our needs.

### All-in-One
**None exist in Go, Rust, or C.** This is the gap go-media fills.

## Language Decision: Go

Rust and C were evaluated. Conclusion: **Go is optimal** because:
1. Heavy work is C/C++ (FFmpeg, whisper.cpp) regardless of wrapper language
2. ox-whisper already deployed at `:8092` — HTTP overhead ~15ms/chunk vs 2s inference = negligible
3. Go ecosystem already in place (go-threads, go-stealth, go-stt)
4. Native extractors (go-threads, kkdai/youtube) eliminate Python dependency for key platforms
5. Development speed: ~3x faster than Rust for same result

## Architecture

### Core Interfaces

```go
// Extractor fetches media metadata from a platform URL.
type Extractor interface {
    Name() string
    Match(url string) bool
    Extract(ctx context.Context, url string) (*Media, error)
}

// Transcriber converts audio to text.
type Transcriber interface {
    Transcribe(ctx context.Context, audioPath string) (*Transcription, error)
    Available() bool
}

// Processor orchestrates: extract → download → audio → transcribe.
type Processor struct { ... }
```

### Core Types

```go
type Media struct {
    Platform    string
    URL         string
    VideoURL    string
    Title       string
    Description string
    Duration    time.Duration
    Qualities   []Quality
    Metadata    map[string]string
}

type Quality struct {
    Label  string // "1080p", "720p"
    URL    string
    Width  int
    Height int
    Size   int64
}

type Transcription struct {
    Text     string
    Language string
    Duration float64
    Chunks   []Chunk
}

type Chunk struct {
    Start float64
    End   float64
    Text  string
}

type Result struct {
    Media         *Media
    VideoPath     string
    Transcription *Transcription
}
```

## API Usage

```go
p := media.NewProcessor(
    media.WithExtractor(instagram.New(threadsClient)),
    media.WithExtractor(youtube.New()),
    media.WithTranscriber(openai.New("http://localhost:8092/v1")),
)

result, err := p.Process(ctx, url, media.Options{
    MaxSize:  50 * 1024 * 1024,
    ChunkSec: 20,
    TempDir:  "/tmp/media",
})
```

## Dependencies

| Dependency | Purpose | New? |
|---|---|---|
| go-threads | Instagram/Threads API | No |
| go-stealth | TLS fingerprinting | No |
| go-stt | STT client | No |
| kkdai/youtube/v2 | YouTube Innertube | Yes |
| ffmpeg/ffprobe | Audio extraction (runtime) | No |

## What Stays in Vaelor

- `InstagramTool` (tool interface wrapper)
- `MediaSender` (Telegram delivery)
- `formatToolResult()` / `formatVideoCaption()` (Telegram formatting)
- Skill instruction injection
