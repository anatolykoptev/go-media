package media_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	media "github.com/anatolykoptev/go-media"
)

func hasFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func hasFFprobe() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// unavailableTranscriber implements media.Transcriber with Available() = false.
type unavailableTranscriber struct{}

func (unavailableTranscriber) Transcribe(_ context.Context, _ string) (*media.Transcription, error) {
	return nil, fmt.Errorf("not available")
}

func (unavailableTranscriber) Available() bool { return false }

func TestProbeDurationNoFile(t *testing.T) {
	dur := media.ProbeDuration(context.Background(), "/tmp/nonexistent_video_file_12345.mp4")
	if dur != 0 {
		t.Errorf("expected 0 for nonexistent file, got %d", dur)
	}
}

func TestExtractAudioChunkNoFFmpeg(t *testing.T) {
	if hasFFmpeg() {
		t.Skip("ffmpeg is available, skipping no-ffmpeg test")
	}

	err := media.ExtractAudioChunk(context.Background(), "input.mp4", "output.wav", 0, 10)
	if err == nil {
		t.Error("expected error when ffmpeg is not in PATH")
	}
}

func TestChunkAndTranscribeNilTranscriber(t *testing.T) {
	result := media.ChunkAndTranscribe(context.Background(), "video.mp4", t.TempDir(), nil, media.Options{})
	if result != nil {
		t.Error("expected nil result for nil transcriber")
	}
}

func TestChunkAndTranscribeUnavailable(t *testing.T) {
	tr := unavailableTranscriber{}
	result := media.ChunkAndTranscribe(context.Background(), "video.mp4", t.TempDir(), tr, media.Options{})
	if result != nil {
		t.Error("expected nil result for unavailable transcriber")
	}
}

func TestExtractAudioChunkIntegration(t *testing.T) {
	if !hasFFmpeg() || !hasFFprobe() {
		t.Skip("ffmpeg/ffprobe not available")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()
	videoPath := filepath.Join(tmpDir, "silent.mp4")

	// Generate a 3-second silent video with ffmpeg.
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi", "-i", "color=c=black:s=320x240:d=3",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-t", "3",
		"-c:v", "libx264", "-c:a", "aac",
		"-shortest", "-y", videoPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}

	// Probe duration — expect 3 or 4 (rounds up).
	dur := media.ProbeDuration(ctx, videoPath)
	if dur < 3 || dur > 4 {
		t.Errorf("expected duration 3-4, got %d", dur)
	}

	// Extract audio chunk.
	chunkPath := filepath.Join(tmpDir, "chunk.wav")
	if err := media.ExtractAudioChunk(ctx, videoPath, chunkPath, 0, 3); err != nil {
		t.Fatalf("ExtractAudioChunk failed: %v", err)
	}

	info, err := os.Stat(chunkPath)
	if err != nil {
		t.Fatalf("chunk file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("chunk file is empty")
	}
}
