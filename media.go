// Package media provides a pipeline for downloading videos from social platforms,
// extracting audio, and transcribing speech.
package media

import "time"

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
}

// MediaStats holds engagement metrics for a media post.
type MediaStats struct {
	Views    int64 // view/play count
	Likes    int64 // like/heart count
	Comments int64 // comment/reply count
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
	VideoPath     string         // path to downloaded video file
	Transcription *Transcription // transcription result (nil if not requested or no speech)
	VideoClips    []VideoClip    // extracted video clips (nil if ExtractClips option not set)
}
