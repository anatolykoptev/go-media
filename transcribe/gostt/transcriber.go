// Package gostt provides a Transcriber that wraps the go-kit voice/stt client.
package gostt

import (
	"context"
	"fmt"

	stt "github.com/anatolykoptev/go-kit/voice/stt"

	"github.com/anatolykoptev/go-media"
)

// Transcriber wraps a go-kit voice/stt Client for OpenAI-compatible STT services.
type Transcriber struct {
	client *stt.Client
}

// Option configures the Transcriber.
type Option func(*stt.Client)

// New creates a Transcriber using go-kit voice/stt.
// baseURL is the STT service endpoint (e.g. "http://localhost:8092").
func New(baseURL string, opts ...stt.Option) *Transcriber {
	return &Transcriber{
		client: stt.New(baseURL, opts...),
	}
}

func (t *Transcriber) Transcribe(ctx context.Context, audioPath string) (*media.Transcription, error) {
	resp, err := t.client.Transcribe(ctx, audioPath)
	if err != nil {
		return nil, fmt.Errorf("go-stt transcribe: %w", err)
	}
	return &media.Transcription{
		Text:     resp.Text,
		Language: resp.Language,
		Duration: resp.Duration,
	}, nil
}

func (t *Transcriber) Available() bool {
	return t.client.IsAvailable()
}
