# go-media

Universal media processing library for Go: download video from social platforms, extract audio, and transcribe speech.

## Install

```bash
go get github.com/anatolykoptev/go-media
```

## Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/anatolykoptev/go-media"
    "github.com/anatolykoptev/go-media/extract/instagram"
    "github.com/anatolykoptev/go-media/transcribe/gostt"
    threads "github.com/anatolykoptev/go-threads"
)

func main() {
    // Set up platform client
    threadsClient, _ := threads.NewClient(threads.Config{})

    // Create processor with extractors and transcriber
    p := media.NewProcessor(
        media.WithExtractor(instagram.New(threadsClient)),
        media.WithTranscriber(gostt.New("http://localhost:8092")),
    )

    // Process a video URL
    result, err := p.Process(context.Background(), "https://instagram.com/reel/ABC123", media.Options{
        MaxSize:  50 * 1024 * 1024, // 50 MB limit
        ChunkSec: 20,               // 20s audio chunks for Whisper
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Video: %s\n", result.VideoPath)
    if result.Transcription != nil {
        fmt.Printf("Text: %s\n", result.Transcription.Text)
    }
}
```

## Supported Platforms

| Platform | Status | Package |
|----------|--------|---------|
| Instagram/Threads | ✅ | `extract/instagram` |
| YouTube | ✅ | `extract/youtube` |
| Twitter/X | Planned | `extract/twitter` |
| Reddit | Planned | `extract/reddit` |
| Telegram | Planned | `extract/telegram` |
| TikTok | Planned | `extract/tiktok` |
| VK | Planned | `extract/vk` |

## Transcription Backends

| Backend | Package |
|---------|---------|
| Any OpenAI-compatible speech-to-text service, through the go-stt client | `transcribe/gostt` |

Beyond `Process`, the root package exports the building blocks: `Extract` (resolve a URL to a
`Media` without downloading), `DownloadFile`, `ExtractAudioChunk`, `ChunkAndTranscribe`,
`ExtractVideoClip` and `ExtractVideoClipsFromChunks`, `MergeDASH` and `MergeAudioVideo`, and
`ProbeDuration`. The `extract/dash` package parses MPEG-DASH manifests and picks the best
representation pair that fits a byte budget. Carousel and photo posts are returned as `Slides`
without transcription.

## Requirements

- Go 1.26+
- `ffmpeg` and `ffprobe` in PATH (for audio extraction)

## Status

Pre-1.0 (`v0.x`); the API can change between minor versions, so pin a version. Versions and
[CHANGELOG.md](CHANGELOG.md) come from release-please. The library reads no environment variables;
everything is configured through options.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md); coding agents start at [CLAUDE.md](CLAUDE.md). Every
document is listed in [docs/INDEX.md](docs/INDEX.md).

## License

Apache 2.0, see [LICENSE](LICENSE).
