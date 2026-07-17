package media

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// audioFadeFilter returns an ffmpeg afade filter string that applies a short
// fade-in at the start and fade-out at the end of a segment. This prevents
// audible clicks/pops at cut boundaries where the waveform is not at a
// zero-crossing. Reference: ffmpego internal/ffutil/fade.go.
func audioFadeFilter(segmentDuration, fadeDuration float64) string {
	if fadeDuration <= 0 {
		fadeDuration = DefaultFadeDurationSec
	}
	fadeOutStart := segmentDuration - fadeDuration
	if fadeOutStart < 0 {
		fadeOutStart = 0
	}
	return fmt.Sprintf("afade=t=in:d=%.3f,afade=t=out:st=%.3f:d=%.3f",
		fadeDuration, fadeOutStart, fadeDuration)
}

// ExtractVideoClip extracts a video segment from startSec to endSec using ffmpeg.
// Video stream is copied (-c:v copy) for fast, lossless extraction. Audio gets
// a short fade-in/fade-out to prevent clicks at cut boundaries.
// The -ss flag is placed before -i for fast keyframe-level seeking.
func ExtractVideoClip(ctx context.Context, videoPath, outputPath string, startSec, endSec float64) error {
	ffCtx, cancel := context.WithTimeout(ctx, DefaultFFmpegTimeout)
	defer cancel()

	duration := endSec - startSec
	if duration <= 0 {
		return fmt.Errorf("invalid clip range: start %.3f >= end %.3f", startSec, endSec)
	}

	fadeFilter := audioFadeFilter(duration, DefaultFadeDurationSec)

	cmd := exec.CommandContext(ffCtx, "ffmpeg", //nolint:gosec // file paths are library-controlled, not user input
		"-ss", fmt.Sprintf("%.3f", startSec),
		"-i", videoPath,
		"-t", fmt.Sprintf("%.3f", duration),
		"-c:v", "copy",
		"-af", fadeFilter,
		"-y",
		outputPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg extract clip %.1f-%.1f: %w", startSec, endSec, err)
	}
	return nil
}

// ExtractVideoClipsFromChunks extracts video clips for each transcription chunk.
// Clips are named "<base>_clip_<index>.mp4" in tempDir. Chunks with empty text
// are skipped. Each clip is extended by paddingSec before and after the chunk
// boundary (clamped to [0, totalDuration]) to avoid cutting mid-word.
// Returns the extracted clips and a count of failures.
func ExtractVideoClipsFromChunks(ctx context.Context, videoPath, tempDir string, chunks []Chunk, paddingSec, totalDuration float64) ([]VideoClip, int) {
	base := filepath.Base(videoPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var clips []VideoClip
	var failed int

	for i, ch := range chunks {
		if strings.TrimSpace(ch.Text) == "" {
			continue
		}

		// Apply padding and clamp to valid range.
		clipStart := ch.Start - paddingSec
		if clipStart < 0 {
			clipStart = 0
		}
		clipEnd := ch.End + paddingSec
		if totalDuration > 0 && clipEnd > totalDuration {
			clipEnd = totalDuration
		}

		clipPath := filepath.Join(tempDir, fmt.Sprintf("%s_clip_%d.mp4", name, i))
		if err := ExtractVideoClip(ctx, videoPath, clipPath, clipStart, clipEnd); err != nil {
			cleanupFile(clipPath)
			failed++
			continue
		}
		clips = append(clips, VideoClip{
			Path:  clipPath,
			Start: clipStart,
			End:   clipEnd,
			Text:  ch.Text,
		})
	}

	return clips, failed
}
