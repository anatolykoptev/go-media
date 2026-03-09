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
    "github.com/anatolykoptev/go-media/transcribe/openai"
    threads "github.com/anatolykoptev/go-threads"
)

func main() {
    // Set up platform client
    threadsClient, _ := threads.NewClient(threads.Config{})

    // Create processor with extractors and transcriber
    p := media.NewProcessor(
        media.WithExtractor(instagram.New(threadsClient)),
        media.WithTranscriber(openai.New("http://localhost:8092/v1")),
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
| OpenAI-compatible (ox-whisper, Groq) | `transcribe/openai` |
| go-stt | `transcribe/gostt` |

## Requirements

- Go 1.26+
- `ffmpeg` and `ffprobe` in PATH (for audio extraction)

## License

MIT
