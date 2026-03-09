# go-media Architecture

## Overview

go-media is a pipeline library: **Extract → Download → Audio → Transcribe**.

```
┌─────────┐     ┌──────────┐     ┌─────────┐     ┌────────────┐
│ URL     │────▶│ Extractor│────▶│Download │────▶│ FFmpeg     │
│ (input) │     │ Registry │     │ (HTTP)  │     │ (audio)    │
└─────────┘     └──────────┘     └─────────┘     └────────────┘
                                                       │
                                                       ▼
                                                 ┌────────────┐
                                                 │ Transcriber│
                                                 │ (chunked)  │
                                                 └────────────┘
                                                       │
                                                       ▼
                                                 ┌────────────┐
                                                 │ Result     │
                                                 │ (text+meta)│
                                                 └────────────┘
```

## Package Structure

```
go-media/
├── media.go           # Core types: Media, Quality, Transcription, Chunk, Result
├── extractor.go       # Extractor interface + Registry (Match → Extract)
├── transcriber.go     # Transcriber interface
├── processor.go       # Processor: orchestrates full pipeline
├── download.go        # DownloadFile: HTTP download with timeout + progress
├── audio.go           # FFmpeg: extract audio, probe duration, chunk
├── options.go         # Options, functional options for Processor
│
├── extract/           # Platform extractors (each implements Extractor)
│   ├── instagram/     # go-threads client → Media
│   ├── youtube/       # kkdai/youtube → Media
│   ├── twitter/       # Syndication API → Media
│   ├── reddit/        # DASH video+audio → Media
│   ├── telegram/      # Bot API getFile → Media
│   ├── tiktok/        # ox-browser fetch-smart → Media
│   └── vk/            # VK Video API → Media
│
├── transcribe/        # Transcription backends (each implements Transcriber)
│   ├── openai/        # OpenAI-compatible HTTP (ox-whisper, Groq)
│   └── gostt/         # go-stt wrapper
│
└── docs/
```

## Design Principles

### 1. Explicit Registration (no init() magic)
Consumers register only the extractors they need:
```go
p := media.NewProcessor(
    media.WithExtractor(instagram.New(client)),
    media.WithExtractor(youtube.New()),
)
```
No global registry, no import side-effects. Binary only includes what's used.

### 2. Interface Segregation
- `Extractor` — knows how to get video URL from platform URL
- `Transcriber` — knows how to convert audio to text
- `Processor` — knows how to wire them together
Each can be used independently.

### 3. No Logging, No Global State
Functions return errors. No logger dependency. Consumer decides how to log.
Exception: progress callbacks for long operations (download, transcribe).

### 4. exec.Command for FFmpeg
No FFmpeg Go bindings. Reasons:
- Our needs are simple (extract audio, probe duration, chunk)
- exec.Command is 20ms overhead on 30s video — invisible
- Zero build complexity (no cgo, no cmake)
- FFmpeg binary is ubiquitous in Docker images

### 5. Chunked Transcription
Audio is split into configurable chunks (default 20s) before STT. Reasons:
- whisper.cpp OOM on ARM64 with >25s audio
- Parallel chunk transcription possible
- Progress reporting per chunk

## Data Flow

### Extract Phase
```
URL → Registry.Match(url) → extractor.Extract(ctx, url) → *Media
```
Each extractor parses the platform URL, calls the platform API, returns `Media` with `VideoURL` and metadata.

### Download Phase
```
Media.VideoURL → HTTP GET → temp file (with timeout, size limit)
```
Streaming download with configurable max size. Progress callback.

### Audio Phase
```
video.mp4 → ffprobe (duration) → ffmpeg -ss -t (chunks) → chunk_0.wav, chunk_1.wav, ...
```
Output: 16kHz mono PCM WAV files. Chunk duration configurable.

### Transcribe Phase
```
chunk_0.wav → Transcriber.Transcribe() → Chunk{Text, Start, End}
chunk_1.wav → Transcriber.Transcribe() → Chunk{Text, Start, End}
...
→ join chunks → Transcription{Text, Chunks}
```
Sequential by default. Parallel option available.

## Error Handling

- Platform API errors → wrap with platform name: `"instagram: rate limited"`
- Download errors → wrap with URL context
- FFmpeg errors → return stderr content
- Transcription errors → per-chunk: skip failed chunks, return partial result
- All errors use `fmt.Errorf("context: %w", err)` for unwrapping

## Concurrency

- `Processor` is safe for concurrent use (stateless per call)
- `Extractor` implementations must be safe for concurrent use
- `Transcriber` implementations must be safe for concurrent use
- Chunk transcription: sequential by default, `Options.Parallel` for concurrent

## Testing Strategy

- Unit tests per package (mock HTTP, mock ffmpeg output)
- Integration tests with `//go:build integration` tag
- No live API tests in CI — only with explicit credentials
- FFmpeg tests skip if `ffmpeg` not in PATH
