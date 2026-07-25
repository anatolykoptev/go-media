// Package media provides a pipeline for downloading videos from social platforms,
// extracting audio, and transcribing speech.
package media

import (
	"fmt"
	"time"
)

// Media represents extracted metadata and download information for a video.
type Media struct {
	Platform    string            // platform name: "instagram", "youtube", etc.
	URL         string            // original input URL
	VideoURL    string            // direct video CDN URL
	AudioURL    string            // separate audio URL (for DASH merge)
	LocalPath   string            // path to already-downloaded file (skips download)
	Title       string            // post/video title
	Description string            // post caption or video description
	Author      string            // author display name or @username
	Duration    time.Duration     // video duration (zero if unknown)
	Qualities   []Quality         // available quality options
	Stats       MediaStats        // engagement stats (likes, views, etc.)
	Metadata    map[string]string // platform-specific key-value pairs
	// Slides is the ordered, per-slide view of a carousel (or a single photo
	// post). Empty for a single video, which uses VideoURL/AudioURL above so
	// the single-video pipeline (transcription, DASH mux, clips) is unchanged.
	// Each Slide carries the CDN URL chosen for that slide by the extractor's
	// rendition selection (best candidate per slide — the same H.264-only DASH
	// rule as a single video for video slides, highest-resolution candidate for
	// photo slides). The processor downloads every slide and reports the local
	// paths in Result.Slides, in the same order.
	Slides []Slide
}

// SlideType is the per-slide media kind of a carousel slide.
type SlideType int

const (
	// SlideTypeImage is a photo slide (go-threads CarouselItem.MediaType 1).
	SlideTypeImage SlideType = 1
	// SlideTypeVideo is a video slide (go-threads CarouselItem.MediaType 2).
	SlideTypeVideo SlideType = 2
)

// Slide is one ordered item of a carousel (or a single photo post),
// carrying the CDN URL chosen for it by the extractor. For a video slide
// that went through DASH, AudioURL carries the separate audio stream URL
// (the processor muxes it); empty for photo slides and for video slides
// that fell back to a self-contained video_versions rendition.
type Slide struct {
	Type     SlideType
	URL      string // chosen CDN URL for this slide
	AudioURL string // separate audio URL (DASH video slides only)
	Width    int
	Height   int
}

// MediaStats holds engagement metrics for a media post.
type MediaStats struct {
	Views    int64 // view/play count
	Likes    int64 // like/heart count
	Comments int64 // comment/reply count
	Shares   int64 // share/repost count
}

// Quality represents a single video quality variant.
type Quality struct {
	Label  string // human label: "1080p", "720p", "360p"
	URL    string // direct download URL for this quality
	Width  int    // pixels, 0 if unknown
	Height int    // pixels, 0 if unknown
	Size   int64  // estimated bytes, 0 if unknown
}

// Transcription holds the result of speech-to-text processing.
type Transcription struct {
	Text         string  // full concatenated text
	Language     string  // detected language code (e.g. "en", "ru")
	Duration     float64 // audio duration in seconds
	Chunks       []Chunk // per-segment results with timestamps
	FailedChunks int     // number of chunks that failed extraction or transcription
}

// Chunk represents a single transcribed audio segment.
type Chunk struct {
	Start float64 // segment start time in seconds
	End   float64 // segment end time in seconds
	Text  string  // transcribed text for this segment
}

// VideoClip is a short video segment extracted from the downloaded video,
// corresponding to a transcription chunk. The caller is responsible for
// cleaning up the clip files (Path) after use.
type VideoClip struct {
	Path  string  // path to the extracted clip file
	Start float64 // clip start time in seconds
	End   float64 // clip end time in seconds
	Text  string  // transcription text for this clip
}

// Result is the output of a full processing pipeline.
type Result struct {
	Media         *Media         // extracted media metadata
	VideoPath     string         // path to downloaded video file (single-video path)
	Transcription *Transcription // transcription result (nil if not requested or no speech)
	VideoClips    []VideoClip    // extracted video clips (nil if ExtractClips option not set)
	// Slides is the per-slide download outcome for a carousel / photo post,
	// in slide order. It always has one entry per expected slide (len ==
	// len(Media.Slides)); a failed slide has an empty Path and a non-nil Err.
	// Empty for the single-video path, which uses VideoPath above. When any
	// slide fails, Process returns this Result together with a *SlideError so
	// the caller can detect partial success without ranging the slice.
	Slides []SlideResult
}

// SlideResult is the download outcome for one carousel slide.
type SlideResult struct {
	Index int // slide position in the album (0-based, matches Media.Slides order)
	Type  SlideType
	Path  string // local file path; empty if this slide failed to download
	Err   error  // non-nil if this slide failed; nil on success
}

// SlideError reports that one or more carousel slides failed to download
// while the post as a whole was extracted. It is returned by Process together
// with a partial Result (whose Slides slice carries the per-slide detail).
// Expected is the total slide count; Succeeded is how many downloaded; Failed
// lists the 0-based indices that did not. A caller can distinguish a total
// slide failure (Succeeded == 0) from a partial one (Succeeded > 0) without
// ranging Result.Slides, and can read each failure's cause from
// Result.Slides[i].Err.
type SlideError struct {
	Expected  int
	Succeeded int
	Failed    []int
}

func (e *SlideError) Error() string {
	return fmt.Sprintf("slides: %d of %d failed: %v", len(e.Failed), e.Expected, e.Failed)
}
