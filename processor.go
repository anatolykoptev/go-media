package media

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

const (
	tempDirPerm   = 0o750
	hashMultipler = 31
)

// Processor orchestrates the full media pipeline: extract → download → transcribe.
type Processor struct {
	registry    *Registry
	transcriber Transcriber
	httpClient  HTTPDoer
}

// NewProcessor creates a Processor with the given options.
func NewProcessor(opts ...ProcessorOption) *Processor {
	p := &Processor{
		registry:   NewRegistry(),
		httpClient: http.DefaultClient,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// Extract finds the matching extractor and returns media metadata.
func (p *Processor) Extract(ctx context.Context, url string) (*Media, error) {
	return p.registry.Extract(ctx, url)
}

// Process runs the full pipeline: extract metadata → download video → transcribe audio.
func (p *Processor) Process(ctx context.Context, url string, opts Options) (*Result, error) {
	opts.defaults()

	// Extract media metadata
	m, err := p.registry.Extract(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}

	if m.VideoURL == "" {
		return nil, fmt.Errorf("no video URL found for %s", url)
	}

	// Prepare temp directory
	tempDir := opts.TempDir
	if tempDir == "" {
		tempDir = filepath.Join(os.TempDir(), "go-media")
	}
	if err := os.MkdirAll(tempDir, tempDirPerm); err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	// Download video
	videoPath := filepath.Join(tempDir, fmt.Sprintf("%s_%s.mp4", m.Platform, sanitizeFilename(url)))

	if err := DownloadFile(ctx, p.httpClient, m.VideoURL, videoPath, opts.MaxSize); err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	// Transcribe (optional)
	var transcription *Transcription
	if p.transcriber != nil {
		transcription = ChunkAndTranscribe(ctx, videoPath, tempDir, p.transcriber, opts)
	}

	return &Result{
		Media:         m,
		VideoPath:     videoPath,
		Transcription: transcription,
	}, nil
}

// Platforms returns names of all registered extractors.
func (p *Processor) Platforms() []string {
	return p.registry.Platforms()
}

// sanitizeFilename creates a safe filename from a URL.
func sanitizeFilename(url string) string {
	h := uint32(0)
	for _, c := range url {
		h = h*hashMultipler + uint32(c)
	}
	return fmt.Sprintf("%08x", h)
}
