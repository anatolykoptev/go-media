package media_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	media "github.com/anatolykoptev/go-media"
)

func TestExtractVideoClipInvalidRange(t *testing.T) {
	err := media.ExtractVideoClip(context.Background(), "input.mp4", "out.mp4", 5.0, 5.0)
	if err == nil {
		t.Error("expected error for start >= end")
	}

	err = media.ExtractVideoClip(context.Background(), "input.mp4", "out.mp4", 10.0, 5.0)
	if err == nil {
		t.Error("expected error for start > end")
	}
}

func TestExtractVideoClipNoFFmpeg(t *testing.T) {
	if hasFFmpeg() {
		t.Skip("ffmpeg is available, skipping no-ffmpeg test")
	}

	err := media.ExtractVideoClip(context.Background(), "input.mp4", "out.mp4", 0, 5)
	if err == nil {
		t.Error("expected error when ffmpeg is not in PATH")
	}
}

func TestExtractVideoClipIntegration(t *testing.T) {
	if !hasFFmpeg() || !hasFFprobe() {
		t.Skip("ffmpeg/ffprobe not available")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()
	videoPath := filepath.Join(tmpDir, "source.mp4")

	// Generate a 5-second silent video with ffmpeg.
	genCmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi", "-i", "color=c=blue:s=320x240:d=5",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-t", "5",
		"-c:v", "libx264", "-c:a", "aac",
		"-shortest", "-y", videoPath,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}

	// Extract a 2-second clip from 1.0 to 3.0.
	clipPath := filepath.Join(tmpDir, "clip.mp4")
	if err := media.ExtractVideoClip(ctx, videoPath, clipPath, 1.0, 3.0); err != nil {
		t.Fatalf("ExtractVideoClip failed: %v", err)
	}

	info, err := os.Stat(clipPath)
	if err != nil {
		t.Fatalf("clip file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("clip file is empty")
	}

	// Verify clip duration is ~2 seconds.
	dur, err := media.ProbeDuration(ctx, clipPath)
	if err != nil {
		t.Fatalf("ProbeDuration on clip failed: %v", err)
	}
	if dur < 2 || dur > 3 {
		t.Errorf("expected clip duration 2-3s, got %d", dur)
	}
}

func TestExtractVideoClipsFromChunksSkipsEmptyText(t *testing.T) {
	if !hasFFmpeg() {
		t.Skip("ffmpeg not available")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()
	videoPath := filepath.Join(tmpDir, "source.mp4")

	genCmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi", "-i", "color=c=red:s=160x120:d=4",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-t", "4",
		"-c:v", "libx264", "-c:a", "aac",
		"-shortest", "-y", videoPath,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}

	// Chunks: first has text, second is empty (should be skipped), third has text.
	chunks := []media.Chunk{
		{Start: 0, End: 1, Text: "hello"},
		{Start: 1, End: 2, Text: ""},
		{Start: 2, End: 3, Text: "world"},
	}

	clips, failed := media.ExtractVideoClipsFromChunks(ctx, videoPath, tmpDir, chunks)
	if failed != 0 {
		t.Errorf("expected 0 failed clips, got %d", failed)
	}
	if len(clips) != 2 {
		t.Fatalf("expected 2 clips (empty text skipped), got %d", len(clips))
	}
	if clips[0].Text != "hello" {
		t.Errorf("expected first clip text 'hello', got %q", clips[0].Text)
	}
	if clips[1].Text != "world" {
		t.Errorf("expected second clip text 'world', got %q", clips[1].Text)
	}

	// Verify clip files exist.
	for _, c := range clips {
		if _, err := os.Stat(c.Path); err != nil {
			t.Errorf("clip file not found: %s: %v", c.Path, err)
		}
	}
}
