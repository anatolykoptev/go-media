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
	// spent tracks retained bytes across the album so MaxTotalSize bounds the
	// call, not just each slide (an N-slide album at only a per-file cap can
	// reach N x MaxSize).
	var spent int64
	for i, slide := range m.Slides {
		sr, n := p.processSlide(ctx, slide, i, tempDir, base, opts, spent)
		spent += n
		if sr.Err == nil {
			succeeded++
		}
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

// processSlide downloads one slide — with the DASH mux for video slides —
// under the remaining call budget and returns its outcome plus the retained
// byte count of the surviving artifact (0 on failure).
func (p *Processor) processSlide(ctx context.Context, slide Slide, i int, tempDir, base string, opts Options, spent int64) (SlideResult, int64) {
	sr := SlideResult{Index: i, Type: slide.Type}
	if slide.URL == "" {
		sr.Err = fmt.Errorf("slide %d: no URL", i)
		return sr, 0
	}
	perFile := opts.remainingCap(spent)
	if opts.MaxTotalSize > 0 && perFile <= 0 {
		sr.Err = fmt.Errorf("slide %d: download: per-call budget exhausted (%d bytes)", i, opts.MaxTotalSize)
		return sr, 0
	}
	path := filepath.Join(tempDir, fmt.Sprintf("%s_slide%d%s", base, i, slideExt(slide.Type)))
	if err := DownloadFile(ctx, p.httpClient, slide.URL, path, perFile); err != nil {
		sr.Err = fmt.Errorf("slide %d: download: %w", i, err)
		return sr, 0
	}
	// Video slide DASH mux — same mergeDASH the single-video path uses.
	if slide.Type == SlideTypeVideo && slide.AudioURL != "" {
		audioCap := opts.remainingCap(spent + fileSize(path))
		if opts.MaxTotalSize > 0 && audioCap <= 0 {
			sr.Err = fmt.Errorf("slide %d: dash merge: per-call budget exhausted (%d bytes)", i, opts.MaxTotalSize)
			_ = os.Remove(path) //nolint:errcheck // best-effort cleanup
			return sr, 0
		}
		merged, err := p.mergeDASH(ctx, path, slide.AudioURL, audioCap)
		if err != nil {
			sr.Err = fmt.Errorf("slide %d: dash merge: %w", i, err)
			_ = os.Remove(path) //nolint:errcheck // best-effort cleanup
			return sr, 0
		}
		path = merged
	}
	sr.Path = path
	return sr, fileSize(path)
}

// slideExt returns the file extension for a slide's downloaded file.
func slideExt(t SlideType) string {
	if t == SlideTypeVideo {
		return ".mp4"
	}
	return ".jpg"
}

// fileSize returns the byte size of path, 0 on any stat error — a missing or
// unreadable file contributes nothing to the call budget.
func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}
