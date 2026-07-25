package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// processSlides downloads every slide of a carousel / photo post and returns a
// Result whose Slides slice carries the local path (or per-slide error) for
// each slide, in order. Transcription is NOT run for slides — whether video
// slides get transcribed is the consumer's call for now; this path only makes
// the local files available. opts.MaxSize bounds each slide independently (see
// Options.MaxSize). When any slide fails, the Result is still returned together
// with a *SlideError so the caller can detect partial success; the failed
// slides have an empty Path and a non-nil Err, the successful ones their Path.
func (p *Processor) processSlides(ctx context.Context, m *Media, opts Options) (*Result, error) {
	tempDir := opts.TempDir
	if tempDir == "" {
		tempDir = filepath.Join(os.TempDir(), "go-media")
	}
	if err := os.MkdirAll(tempDir, tempDirPerm); err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	base := sanitizeFilename(m.URL)
	results := make([]SlideResult, len(m.Slides))
	succeeded := 0
	for i, slide := range m.Slides {
		sr := SlideResult{Index: i, Type: slide.Type}
		if slide.URL == "" {
			sr.Err = fmt.Errorf("slide %d: no URL", i)
			results[i] = sr
			continue
		}
		path := filepath.Join(tempDir, fmt.Sprintf("%s_slide%d%s", base, i, slideExt(slide.Type)))
		if err := DownloadFile(ctx, p.httpClient, slide.URL, path, opts.MaxSize); err != nil {
			sr.Err = fmt.Errorf("slide %d: download: %w", i, err)
			results[i] = sr
			continue
		}
		// Video slide DASH mux — same mergeDASH the single-video path uses.
		if slide.Type == SlideTypeVideo && slide.AudioURL != "" {
			merged, err := p.mergeDASH(ctx, path, slide.AudioURL, opts.MaxSize)
			if err != nil {
				sr.Err = fmt.Errorf("slide %d: dash merge: %w", i, err)
				results[i] = sr
				_ = os.Remove(path) //nolint:errcheck // best-effort cleanup
				continue
			}
			path = merged
		}
		sr.Path = path
		succeeded++
		results[i] = sr
	}

	res := &Result{Media: m, Slides: results}
	if succeeded < len(m.Slides) {
		failed := make([]int, 0, len(m.Slides)-succeeded)
		for i, r := range results {
			if r.Err != nil {
				failed = append(failed, i)
			}
		}
		return res, &SlideError{Expected: len(m.Slides), Succeeded: succeeded, Failed: failed}
	}
	return res, nil
}

// slideExt returns the file extension for a slide's downloaded file.
func slideExt(t SlideType) string {
	if t == SlideTypeVideo {
		return ".mp4"
	}
	return ".jpg"
}
