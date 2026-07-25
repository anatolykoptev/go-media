package media_test

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anatolykoptev/go-media"
)

// stubTranscriber is a deterministic Transcriber for regression tests: it
// returns a fixed *Transcription (success) or a fixed error, with no network
// or model. Available() is always true so ChunkAndTranscribe reaches Transcribe.
type stubTranscriber struct {
	text string
	err  error
}

func (s *stubTranscriber) Available() bool { return true }

func (s *stubTranscriber) Transcribe(_ context.Context, _ string) (*media.Transcription, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &media.Transcription{Text: s.text}, nil
}

// requireFFmpeg skips the test if ffmpeg or ffprobe is not on PATH.
// ChunkAndTranscribe runs both, so a stub transcriber alone cannot reach the
// Transcribe call without them.
func requireFFmpeg(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not in PATH", bin)
		}
	}
}

// makeTinyVideo writes a 1-second black + silent-audio MP4 at path.
// ChunkAndTranscribe needs a real container ffprobe can parse and a real audio
// stream ffmpeg can extract; a fake byte body fails at ProbeDuration.
func makeTinyVideo(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=160x120:d=1",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-shortest",
		path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg generate test video: %v\n%s", err, out)
	}
}

// TestProcessTranscriptionSurvivesToCaller is the regression test for the
// processSingleVideo transcribe shadow: with the bug, the inner := declared a
// new transcription scoped to the if body, so the outer one stayed nil and
// Result.Transcription was always nil on the success path — the transcript was
// computed and then thrown away. Fails on 7eee415, passes after the := -> = fix.
func TestProcessTranscriptionSurvivesToCaller(t *testing.T) {
	requireFFmpeg(t)
	ctx := context.Background()
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "src.mp4")
	makeTinyVideo(t, ctx, videoPath)

	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: &media.Media{
			Platform: "test", LocalPath: videoPath,
		}}),
		media.WithTranscriber(&stubTranscriber{text: "stub transcribed text"}),
	)
	// ChunkSec large enough that ProbeDuration's int(dur)+1 rounding yields a
	// single chunk — avoids a phantom second chunk that fails extraction and
	// turns the success path into a partial-error return.
	res, err := p.Process(ctx, "https://test.com/p/1",
		media.Options{TempDir: tmp, ChunkSec: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Transcription == nil {
		t.Fatal("Result.Transcription nil — transcript computed but dropped (shadow bug)")
	}
	if !strings.Contains(res.Transcription.Text, "stub transcribed text") {
		t.Fatalf("Result.Transcription.Text = %q, want it to contain the stub's text",
			res.Transcription.Text)
	}
}

// TestProcessTranscribeErrorReturnsPartialResult pins the partial-result-on-
// error contract: when transcription fails, Process returns BOTH a non-nil
// Result (carrying the downloaded VideoPath) AND the wrapped error. The shadow
// fix must not change this behaviour.
func TestProcessTranscribeErrorReturnsPartialResult(t *testing.T) {
	requireFFmpeg(t)
	ctx := context.Background()
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "src.mp4")
	makeTinyVideo(t, ctx, videoPath)

	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: &media.Media{
			Platform: "test", LocalPath: videoPath,
		}}),
		media.WithTranscriber(&stubTranscriber{err: errors.New("stub transcribe failure")}),
	)
	res, err := p.Process(ctx, "https://test.com/p/1",
		media.Options{TempDir: tmp, ChunkSec: 100})
	if err == nil {
		t.Fatal("expected transcribe error, got nil")
	}
	if res == nil {
		t.Fatal("Result nil on transcribe error, want partial Result")
	}
	if res.VideoPath == "" {
		t.Fatal("partial Result.VideoPath empty, want the downloaded file path")
	}
}

// TestProcessTranscriptionReachesClipExtraction proves the clip-extraction
// guard fires when the transcript survives: with ExtractClips set and the stub
// returning a chunk with text, ExtractVideoClipsFromChunks must be reached and
// produce at least one clip. With the shadow bug, transcription was nil so the
// guard never fired and VideoClips was always empty.
func TestProcessTranscriptionReachesClipExtraction(t *testing.T) {
	requireFFmpeg(t)
	ctx := context.Background()
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "src.mp4")
	makeTinyVideo(t, ctx, videoPath)

	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: &media.Media{
			Platform: "test", LocalPath: videoPath,
		}}),
		media.WithTranscriber(&stubTranscriber{text: "stub clip text"}),
	)
	res, err := p.Process(ctx, "https://test.com/p/1",
		media.Options{TempDir: tmp, ChunkSec: 100, ExtractClips: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Transcription == nil {
		t.Fatal("Transcription nil — guard input, expected non-nil")
	}
	if len(res.Transcription.Chunks) == 0 {
		t.Fatal("Transcription.Chunks empty — guard input, expected >=1 chunk")
	}
	if len(res.VideoClips) == 0 {
		t.Fatal("VideoClips empty — clip-extraction guard did not fire (transcription was nil / shadowed)")
	}
}
