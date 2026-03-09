package media

// Options configures a single Process call.
type Options struct {
	MaxSize  int64  // max video file size in bytes (0 = no limit)
	ChunkSec int    // audio chunk duration for transcription (default 20)
	TempDir  string // directory for temporary files (default os.TempDir())
	Parallel bool   // transcribe chunks in parallel
}

// defaults fills zero-value fields with sensible defaults.
func (o *Options) defaults() {
	if o.ChunkSec <= 0 {
		o.ChunkSec = DefaultChunkSec
	}
}

// ProcessorOption configures a Processor.
type ProcessorOption func(*Processor)

// WithExtractor registers a platform extractor.
func WithExtractor(e Extractor) ProcessorOption {
	return func(p *Processor) {
		p.registry.Register(e)
	}
}

// WithTranscriber sets the transcription backend.
func WithTranscriber(t Transcriber) ProcessorOption {
	return func(p *Processor) {
		p.transcriber = t
	}
}

// WithHTTPClient sets a custom HTTP client for video downloads.
func WithHTTPClient(doer HTTPDoer) ProcessorOption {
	return func(p *Processor) {
		p.httpClient = doer
	}
}
