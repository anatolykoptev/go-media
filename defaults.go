package media

import "time"

// Default configuration values for the media processing pipeline.
const (
	// DefaultDownloadTimeout is the maximum time for downloading a video file.
	DefaultDownloadTimeout = 120 * time.Second

	// DefaultFFmpegTimeout is the maximum time for an ffmpeg operation.
	DefaultFFmpegTimeout = 60 * time.Second

	// DefaultProbeTimeout is the maximum time for ffprobe duration check.
	DefaultProbeTimeout = 10 * time.Second

	// DefaultChunkSec is the default audio chunk duration in seconds for transcription.
	DefaultChunkSec = 20

	// tempDirPerm is the permission mode for temp directories.
	tempDirPerm = 0o750

	// hashMultiplier is the FNV-like hash multiplier for URL-to-filename hashing.
	hashMultiplier = 31
)
