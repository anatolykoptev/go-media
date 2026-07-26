package media_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	media "github.com/anatolykoptev/go-media"
)

// requireFFmpeg lives in audio_test.go (shared helper — fail loudly, never
// skip). The mux path needs ffmpeg; the shared helper also requires ffprobe,
// which is always co-installed with ffmpeg and is a harmless strengthening.

// genVideoOnly writes a video-only MP4 (no audio stream) at path using ffmpeg.
func genVideoOnly(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi", "-i", "color=c=black:s=320x240:d=2",
		"-c:v", "libx264", "-an", "-y", path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate video-only file: %v\n%s", err, out)
	}
}

// genAudioM4A writes a 2-second silent AAC/m4a audio file at path using ffmpeg.
func genAudioM4A(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-t", "2", "-c:a", "aac", "-y", path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate audio file: %v\n%s", err, out)
	}
}

// audioServer serves the bytes of srcPath as the response body for every
// request, so MergeDASH's DownloadFile fetches a real audio file.
func audioServer(t *testing.T, srcPath string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b, err := os.ReadFile(srcPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "audio/mp4")
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestMergeDASHDirectMuxesAndReplacesVideoPath calls MergeDASH directly (not
// via Processor) and asserts the merged result is left at videoPath: the
// original video-only file is replaced in place by the muxed file, and the
// returned path equals videoPath. ffmpeg is a hard precondition — the test
// FAILs (never skips) when it is missing.
func TestMergeDASHDirectMuxesAndReplacesVideoPath(t *testing.T) {
	requireFFmpeg(t)
	ctx := context.Background()
	tmp := t.TempDir()

	videoPath := filepath.Join(tmp, "slide.mp4")
	genVideoOnly(t, ctx, videoPath)

	audioSrc := filepath.Join(tmp, "audio.m4a")
	genAudioM4A(t, ctx, audioSrc)
	srv := audioServer(t, audioSrc)

	got, err := media.MergeDASH(ctx, srv.Client(), videoPath, srv.URL+"/audio.m4a", 0)
	if err != nil {
		t.Fatalf("MergeDASH returned error: %v", err)
	}
	if got != videoPath {
		t.Fatalf("returned path = %q, want videoPath %q (merged must rename over original)", got, videoPath)
	}

	// videoPath now holds the muxed file: it must exist and be non-empty,
	// and the intermediate .merged.mp4 / .audio.m4a side files must be gone.
	info, err := os.Stat(videoPath)
	if err != nil {
		t.Fatalf("merged video file missing at videoPath: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("merged video file is empty")
	}
	if _, err := os.Stat(videoPath + ".merged.mp4"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("leftover .merged.mp4: stat err=%v, want ErrNotExist", err)
	}
	if _, err := os.Stat(videoPath + ".audio.m4a"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("leftover .audio.m4a: stat err=%v, want ErrNotExist", err)
	}
}

// TestMergeDASHDownloadFailureReturnsVideoPathIntact: when the audio download
// fails, MergeDASH must return (videoPath, err) with the original video file
// left intact on disk and no audio temp file left behind.
func TestMergeDASHDownloadFailureReturnsVideoPathIntact(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	videoPath := filepath.Join(tmp, "slide.mp4")
	original := []byte("original-video-bytes")
	if err := os.WriteFile(videoPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	got, err := media.MergeDASH(ctx, srv.Client(), videoPath, srv.URL+"/audio.m4a", 0)
	if err == nil {
		t.Fatal("expected error for failed audio download, got nil")
	}
	if got != videoPath {
		t.Fatalf("returned path = %q, want videoPath %q on download failure", got, videoPath)
	}

	got2, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatalf("video file missing after download failure: %v", err)
	}
	if string(got2) != string(original) {
		t.Fatalf("video file mutated on download failure: got %q, want %q", got2, original)
	}
	if _, err := os.Stat(videoPath + ".audio.m4a"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("leftover .audio.m4a after download failure: stat err=%v, want ErrNotExist", err)
	}
}

// TestMergeDASHMuxFailureReturnsVideoPathIntact: when ffmpeg muxing fails
// (audio file is not a valid stream), MergeDASH must return (videoPath, err)
// with the original video file intact and the audio temp file cleaned up.
func TestMergeDASHMuxFailureReturnsVideoPathIntact(t *testing.T) {
	requireFFmpeg(t)
	ctx := context.Background()
	tmp := t.TempDir()

	videoPath := filepath.Join(tmp, "slide.mp4")
	genVideoOnly(t, ctx, videoPath)
	original, _ := os.ReadFile(videoPath)

	// Serve garbage as the "audio" file so ffmpeg mux fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not-an-audio-stream"))
	}))
	defer srv.Close()

	got, err := media.MergeDASH(ctx, srv.Client(), videoPath, srv.URL+"/audio.m4a", 0)
	if err == nil {
		t.Fatal("expected error for mux failure, got nil")
	}
	if got != videoPath {
		t.Fatalf("returned path = %q, want videoPath %q on mux failure", got, videoPath)
	}

	got2, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatalf("video file missing after mux failure: %v", err)
	}
	if string(got2) != string(original) {
		t.Fatalf("video file mutated on mux failure: got %q, want %q", got2, original)
	}
	if _, err := os.Stat(videoPath + ".audio.m4a"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("leftover .audio.m4a after mux failure: stat err=%v, want ErrNotExist", err)
	}
}

// TestMergeDASHRenameFailureContract asserts the rename-failure contract.
//
// RENAME CONTRACT (conclusion): on a rename failure MergeDASH returns
// (mergedPath, err) — NOT videoPath — because the muxed output lives at
// mergedPath and the caller must recover it from there; videoPath is
// indeterminate (os.Remove was already attempted, error ignored). This is
// deliberate and correct, and the consumer (vaelor-agent) ships against it.
// The doc comment on MergeDASH documents this contract.
//
// PORTABILITY LIMITATION: a pure rename-only failure (mux succeeds, rename
// fails) cannot be portably triggered in a unit test because mergedPath =
// videoPath + ".merged.mp4" shares the same directory as videoPath — anything
// that makes os.Rename fail (read-only parent, non-empty-dir destination, full
// filesystem) also makes the mux's mergedPath creation fail first. Triggering
// it requires root (read-only bind mount) or filesystem interposition, neither
// available in a non-root test. The consumer's own test suite
// (vaelor-agent/pkg/media/merge_dash_test.go) does not test this path either.
//
// This test makes videoPath a non-empty directory, which causes the MUX to
// fail (ffmpeg cannot read a directory as input). That exercises the
// mux-failure branch (returns videoPath, err), not the rename branch. It is
// kept here to (a) pin the audio-temp cleanup on this failure mode and (b)
// document the rename contract above — the only portable way to assert it.
func TestMergeDASHRenameFailureContract(t *testing.T) {
	requireFFmpeg(t)
	ctx := context.Background()
	tmp := t.TempDir()

	audioSrc := filepath.Join(tmp, "audio.m4a")
	genAudioM4A(t, ctx, audioSrc)
	srv := audioServer(t, audioSrc)

	// videoPath as a non-empty directory: ffmpeg cannot read it as input,
	// so the mux fails (returns videoPath, err). A pure rename-only failure
	// is not portably triggerable — see the comment above.
	videoPath := filepath.Join(tmp, "slide.mp4")
	if err := os.MkdirAll(videoPath, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(videoPath, "blocker"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := media.MergeDASH(ctx, srv.Client(), videoPath, srv.URL+"/audio.m4a", 0)
	if err == nil {
		t.Fatal("expected error for non-readable videoPath, got nil")
	}
	// Mux-failure branch: returns videoPath (not mergedPath). The
	// rename-failure branch (unreachable here) would return mergedPath.
	if got != videoPath {
		t.Fatalf("RENAME CONTRACT: returned path = %q, want videoPath %q (mux-failure branch; rename-only failure would return mergedPath — see portability comment)", got, videoPath)
	}
	// Audio temp must be cleaned regardless of which branch failed.
	if _, err := os.Stat(videoPath + ".audio.m4a"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("leftover .audio.m4a after failure: stat err=%v, want ErrNotExist", err)
	}
}
