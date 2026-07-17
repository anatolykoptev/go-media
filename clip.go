package media

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtractVideoClip extracts a video segment from startSec to endSec using ffmpeg.
// Uses stream copy (-c copy) for fast, lossless extraction without re-encoding.
// The -ss flag is placed before -i for fast keyframe-level seeking.
func ExtractVideoClip(ctx context.Context, videoPath, outputPath string, startSec, endSec float64) error {
	ffCtx, cancel := context.WithTimeout(ctx, DefaultFFmpegTimeout)
	defer cancel()

	duration := endSec - startSec
	if duration <= 0 {
		return fmt.Errorf("invalid clip range: start %.3f >= end %.3f", startSec, endSec)
	}

	cmd := exec.CommandContext(ffCtx, "ffmpeg", //nolint:gosec // file paths are library-controlled, not user input
		"-ss", fmt.Sprintf("%.3f", startSec),
		"-i", videoPath,
		"-t", fmt.Sprintf("%.3f", duration),
		"-c", "copy",
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
// are skipped. Returns the extracted clips and a count of failures.
func ExtractVideoClipsFromChunks(ctx context.Context, videoPath, tempDir string, chunks []Chunk) ([]VideoClip, int) {
	base := filepath.Base(videoPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var clips []VideoClip
	var failed int

	for i, ch := range chunks {
		if strings.TrimSpace(ch.Text) == "" {
			continue
		}
		clipPath := filepath.Join(tempDir, fmt.Sprintf("%s_clip_%d.mp4", name, i))
		if err := ExtractVideoClip(ctx, videoPath, clipPath, ch.Start, ch.End); err != nil {
			cleanupFile(clipPath)
			failed++
			continue
		}
		clips = append(clips, VideoClip{
			Path:  clipPath,
			Start: ch.Start,
			End:   ch.End,
			Text:  ch.Text,
		})
	}

	return clips, failed
}
