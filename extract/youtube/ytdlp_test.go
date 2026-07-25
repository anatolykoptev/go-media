package youtube

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ytdlp "github.com/lrstanley/go-ytdlp"
)

func TestBuildCommandMaxFileSize(t *testing.T) {
	b := &ytdlpBackend{}
	cfg := Config{}

	t.Run("budget set adds max-filesize flag", func(t *testing.T) {
		cmd := b.buildCommand("/tmp/out.mp4", cfg, 50_000_000)
		flags := cmd.GetFlagConfig().ToFlags().FindByID("max_filesize")
		if len(flags) != 1 {
			t.Fatalf("max_filesize flags: got %d, want 1", len(flags))
		}
		raw := flags[0].Raw()
		// Raw() returns ["--max-filesize=SIZE"] or ["--max-filesize", "SIZE"].
		if len(raw) < 2 {
			t.Fatalf("max_filesize raw args: %v, want >=2", raw)
		}
		// The size argument must be the byte budget (suffixless = bytes).
		if raw[1] != "50000000" {
			t.Fatalf("max_filesize size = %q, want 50000000", raw[1])
		}
	})

	t.Run("zero budget omits max-filesize flag", func(t *testing.T) {
		cmd := b.buildCommand("/tmp/out.mp4", cfg, 0)
		flags := cmd.GetFlagConfig().ToFlags().FindByID("max_filesize")
		if len(flags) != 0 {
			t.Fatalf("max_filesize flags: got %d, want 0 (no budget)", len(flags))
		}
	})
}

// TestDownloadMaxFilesizeSkipReturnsError simulates yt-dlp's --max-filesize
// FILTER behaviour: the download is skipped (no output file written) but the
// process still exits 0. download must surface a clear budget/no-output error
// rather than returning a Media whose LocalPath points at a file that was
// never written (which would make transcription fail on a missing file).
func TestDownloadMaxFilesizeSkipReturnsError(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "out.mp4")

	b := &ytdlpBackend{
		run: func(ctx context.Context, cmd *ytdlp.Command, videoURL string) (*ytdlp.Result, error) {
			// Simulate the filter skip: exit 0, no file written.
			return &ytdlp.Result{ExitCode: 0}, nil
		},
	}

	_, err := b.download(context.Background(), "https://youtube.com/watch?v=abc123def45", outputPath, Config{}, 50_000_000)
	if err == nil {
		t.Fatal("expected error when yt-dlp skipped download, got nil (bogus success)")
	}
	if !strings.Contains(err.Error(), "no output file") {
		t.Fatalf("error = %q, want a no-output/budget error", err.Error())
	}
}

// TestDownloadStatCheckPassesWhenFilePresent confirms the stat guard does NOT
// false-positive on a present, non-empty media file: with the file written the
// guard passes and the (absent) .info.json metadata step is what errors —
// proving the guard is positioned after the run, not before it.
func TestDownloadStatCheckPassesWhenFilePresent(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(outputPath, []byte("media-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	b := &ytdlpBackend{
		run: func(ctx context.Context, cmd *ytdlp.Command, videoURL string) (*ytdlp.Result, error) {
			return &ytdlp.Result{ExitCode: 0}, nil
		},
	}

	_, err := b.download(context.Background(), "https://youtube.com/watch?v=abc123def45", outputPath, Config{}, 0)
	if err == nil {
		t.Fatal("expected metadata error (no info.json sidecar), got nil")
	}
	// Must be a metadata error, NOT the budget/no-output guard firing.
	if strings.Contains(err.Error(), "no output file") {
		t.Fatalf("stat guard fired on a present non-empty file: %v", err)
	}
}
