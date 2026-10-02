package media_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	media "github.com/anatolykoptev/go-media"
)

func TestExtractVideoClipsPaddingAndClamping(t *testing.T) {
	requireFFmpeg(t)

	ctx := context.Background()
	tmpDir := t.TempDir()
	videoPath := filepath.Join(tmpDir, "source.mp4")

	// Generate a 10-second video.
	genCmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi", "-i", "color=c=green:s=160x120:d=10",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-t", "10",
		"-c:v", "libx264", "-c:a", "aac",
		"-shortest", "-y", videoPath,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}

	// Chunks at 2-4 and 8-10 with 1s padding, totalDuration=10.
	// Clip 0: 2-4 → padded to 1-5
	// Clip 1: 8-10 → padded to 7-11 → clamped to 7-10
	chunks := []media.Chunk{
		{Start: 2, End: 4, Text: "first"},
		{Start: 8, End: 10, Text: "last"},
	}

	clips, failed := media.ExtractVideoClipsFromChunks(ctx, videoPath, tmpDir, chunks, 1.0, 10)
	if failed != 0 {
		t.Errorf("expected 0 failed clips, got %d", failed)
	}
	if len(clips) != 2 {
		t.Fatalf("expected 2 clips, got %d", len(clips))
	}

	// Clip 0: padded start=1, end=5
	if clips[0].Start != 1.0 {
		t.Errorf("clip 0 start: expected 1.0, got %.1f", clips[0].Start)
	}
	if clips[0].End != 5.0 {
		t.Errorf("clip 0 end: expected 5.0, got %.1f", clips[0].End)
	}

	// Clip 1: padded start=7, end clamped to 10
	if clips[1].Start != 7.0 {
		t.Errorf("clip 1 start: expected 7.0, got %.1f", clips[1].Start)
	}
	if clips[1].End != 10.0 {
		t.Errorf("clip 1 end: expected 10.0 (clamped), got %.1f", clips[1].End)
	}
}

func TestExtractVideoClipsPaddingClampsToZero(t *testing.T) {
	requireFFmpeg(t)

	ctx := context.Background()
	tmpDir := t.TempDir()
	videoPath := filepath.Join(tmpDir, "source.mp4")

	genCmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi", "-i", "color=c=yellow:s=160x120:d=5",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-t", "5",
		"-c:v", "libx264", "-c:a", "aac",
		"-shortest", "-y", videoPath,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}

	// Chunk at 0-2 with 1s padding → start clamped to 0, end=3.
	chunks := []media.Chunk{
		{Start: 0, End: 2, Text: "beginning"},
	}

	clips, failed := media.ExtractVideoClipsFromChunks(ctx, videoPath, tmpDir, chunks, 1.0, 5)
	if failed != 0 {
		t.Errorf("expected 0 failed clips, got %d", failed)
	}
	if len(clips) != 1 {
		t.Fatalf("expected 1 clip, got %d", len(clips))
	}
	if clips[0].Start != 0 {
		t.Errorf("clip start: expected 0 (clamped), got %.1f", clips[0].Start)
	}
	if clips[0].End != 3.0 {
		t.Errorf("clip end: expected 3.0, got %.1f", clips[0].End)
	}
}
