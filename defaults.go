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

	// DefaultClipPaddingSec is the default padding applied to video clips
	// before and after transcription chunk boundaries, to avoid cutting mid-word.
	DefaultClipPaddingSec = 0.5

	// DefaultFadeDurationSec is the audio fade duration at clip boundaries
	// to prevent clicks/pops from non-zero-crossing cuts.
	DefaultFadeDurationSec = 0.03

	// tempDirPerm is the permission mode for temp directories.
	tempDirPerm = 0o750

	// hashMultiplier is the FNV-like hash multiplier for URL-to-filename hashing.
	hashMultiplier = 31
)
