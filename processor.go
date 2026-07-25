package media

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
// The caller is responsible for cleaning up the temp directory and downloaded files
// after Process returns. If opts.TempDir is empty, files are placed in os.TempDir()/go-media.
func (p *Processor) Process(ctx context.Context, url string, opts Options) (*Result, error) {
	opts.defaults()

	// Extract media metadata, threading the byte budget into budget-aware
	// extractors (DASH representation selection, yt-dlp --max-filesize).
	m, err := p.registry.ExtractWithBudget(ctx, url, opts.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}

	if m.VideoURL == "" && m.LocalPath == "" {
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

	var videoPath string

	if m.LocalPath != "" {
		// Extractor already downloaded the file (e.g. yt-dlp)
		videoPath = m.LocalPath
	} else {
		// Download video
		videoPath = filepath.Join(tempDir, fmt.Sprintf("%s_%s.mp4", m.Platform, sanitizeFilename(url)))
		if err := DownloadFile(ctx, p.httpClient, m.VideoURL, videoPath, opts.MaxSize); err != nil {
			return nil, fmt.Errorf("download: %w", err)
		}

		// DASH: merge separate audio stream if present
		if m.AudioURL != "" {
			videoPath, err = p.mergeDASH(ctx, videoPath, m.AudioURL, opts.MaxSize)
			if err != nil {
				return nil, fmt.Errorf("dash merge: %w", err)
			}
		}
	}

	// Transcribe (optional)
	var transcription *Transcription
	if p.transcriber != nil {
		transcription, err = ChunkAndTranscribe(ctx, videoPath, tempDir, p.transcriber, opts)
		if err != nil {
			// Return partial result with error — consumer can check both.
			return &Result{
				Media:         m,
				VideoPath:     videoPath,
				Transcription: transcription,
			}, fmt.Errorf("transcribe: %w", err)
		}
	}

	// Extract video clips for each transcription chunk (optional)
	var clips []VideoClip
	if opts.ExtractClips && transcription != nil && len(transcription.Chunks) > 0 {
		clips, _ = ExtractVideoClipsFromChunks(ctx, videoPath, tempDir, transcription.Chunks, opts.ClipPaddingSec, transcription.Duration)
		// Clip extraction failures are non-fatal — clips is partial.
	}

	return &Result{
		Media:         m,
		VideoPath:     videoPath,
		Transcription: transcription,
		VideoClips:    clips,
	}, nil
}

// Platforms returns names of all registered extractors.
func (p *Processor) Platforms() []string {
	return p.registry.Platforms()
}

// mergeDASH downloads a separate audio stream and muxes it with the video using ffmpeg.
func (p *Processor) mergeDASH(ctx context.Context, videoPath, audioURL string, maxSize int64) (string, error) {
	audioPath := videoPath + ".audio.m4a"
	if err := DownloadFile(ctx, p.httpClient, audioURL, audioPath, maxSize); err != nil {
		return videoPath, fmt.Errorf("download audio: %w", err) //nolint:wrapcheck // already wrapped
	}
	defer cleanupFile(audioPath)

	mergedPath := videoPath + ".merged.mp4"
	if err := MergeAudioVideo(ctx, videoPath, audioPath, mergedPath); err != nil {
		return videoPath, err
	}

	// Replace original video-only file with merged
	_ = os.Remove(videoPath) //nolint:errcheck // best-effort cleanup
	if err := os.Rename(mergedPath, videoPath); err != nil {
		return mergedPath, fmt.Errorf("rename merged file: %w", err)
	}
	return videoPath, nil
}

// sanitizeFilename creates a safe filename from a URL.
func sanitizeFilename(url string) string {
	h := uint32(0)
	for _, c := range url {
		h = h*hashMultiplier + uint32(c)
	}
	return fmt.Sprintf("%08x", h)
}
