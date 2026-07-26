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

// requireFFmpeg fails the test loudly when ffmpeg or ffprobe is not on PATH.
// Both are required test dependencies, provisioned in CI (see preflight.yml);
// skipping on a missing binary is green-over-skipped — fail instead so an
// under-provisioned environment is reported, not silently tested less.
// The inverse no-ffmpeg tests (TestExtractAudioChunkNoFFmpeg,
// TestExtractVideoClipNoFFmpeg) use hasFFmpeg directly to gate the ABSENCE
// path and must NOT call this.
func requireFFmpeg(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Fatalf("%s not in PATH — required test dependency; install ffmpeg (e.g. `sudo apt-get install -y ffmpeg`), CI provisions it: %v", bin, err)
		}
	}
}

// unavailableTranscriber implements media.Transcriber with Available() = false.
type unavailableTranscriber struct{}

func (unavailableTranscriber) Transcribe(_ context.Context, _ string) (*media.Transcription, error) {
	return nil, fmt.Errorf("not available")
}

func (unavailableTranscriber) Available() bool { return false }

func TestProbeDurationNoFile(t *testing.T) {
	dur, err := media.ProbeDuration(context.Background(), "/tmp/nonexistent_video_file_12345.mp4")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
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
	result, err := media.ChunkAndTranscribe(context.Background(), "video.mp4", t.TempDir(), nil, media.Options{})
	if err != nil {
		t.Errorf("expected nil error for nil transcriber (opt-out), got: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for nil transcriber")
	}
}

func TestChunkAndTranscribeUnavailable(t *testing.T) {
	tr := unavailableTranscriber{}
	result, err := media.ChunkAndTranscribe(context.Background(), "video.mp4", t.TempDir(), tr, media.Options{})
	if err == nil {
		t.Error("expected error for unavailable transcriber")
	}
	if result != nil {
		t.Error("expected nil result for unavailable transcriber")
	}
}

func TestExtractAudioChunkIntegration(t *testing.T) {
	requireFFmpeg(t)

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
	dur, err := media.ProbeDuration(ctx, videoPath)
	if err != nil {
		t.Fatalf("ProbeDuration failed: %v", err)
	}
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
