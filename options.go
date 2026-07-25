package media

// Options configures a single Process call.
type Options struct {
	// MaxSize bounds a single downloaded file in bytes (0 = no limit). For a
	// single video it caps the video (and the separate DASH audio stream). For
	// a carousel / photo post it bounds EACH slide independently — an album of
	// N slides is checked N times, once per slide; the whole-album total is
	// NOT checked. A per-slide cap matches the existing per-file DownloadFile
	// guard and the DASH per-representation budget; a whole-album sum would
	// require downloading every slide first, which the streaming download
	// cannot do mid-flight.
	MaxSize        int64
	ChunkSec       int     // audio chunk duration for transcription (default 20)
	TempDir        string  // directory for temporary files (default os.TempDir())
	ExtractClips   bool    // if true, extract video clips for each transcription chunk
	ClipPaddingSec float64 // seconds to extend clips before/after chunk boundaries (default 0.5)
}

// defaults fills zero-value fields with sensible defaults.
func (o *Options) defaults() {
	if o.ChunkSec <= 0 {
		o.ChunkSec = DefaultChunkSec
	}
	if o.ClipPaddingSec < 0 {
		o.ClipPaddingSec = DefaultClipPaddingSec
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
