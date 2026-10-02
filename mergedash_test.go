package media_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	// The mux must have added the audio: a plain copy of the video-only
	// input would pass every check above.
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "a",
		"-show_entries", "stream=codec_type", "-of", "csv=p=0", videoPath).Output()
	if err != nil {
		t.Fatalf("ffprobe merged file: %v", err)
	}
	if strings.TrimSpace(string(out)) != "audio" {
		t.Fatalf("merged file has no audio stream (ffprobe: %q)", out)
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

// shimFFmpeg puts a fake ffmpeg first on PATH for the rest of the test. The
// script gets MergeAudioVideo's argv (-i VIDEO -i AUDIO -c copy -y OUT), so
// "$2" is the video path and the last argument is the output path.
func shimFFmpeg(t *testing.T, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell shim")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nvideo=\"$2\"\nfor a in \"$@\"; do out=\"$a\"; done\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg"), []byte(script), 0o700); err != nil { //nolint:gosec // test shim must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// dashFixture is a video-only placeholder and an audio server; the shim never
// reads either, so their bytes do not matter.
func dashFixture(t *testing.T) (videoPath, audioURL string, client *http.Client) {
	t.Helper()
	tmp := t.TempDir()
	videoPath = filepath.Join(tmp, "slide.mp4")
	audioSrc := filepath.Join(tmp, "audio.m4a")
	for _, p := range []string{videoPath, audioSrc} {
		if err := os.WriteFile(p, []byte("placeholder"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	srv := audioServer(t, audioSrc)
	return videoPath, srv.URL + "/audio.m4a", srv.Client()
}

// TestMergeDASHRenameFailureReturnsMergedPath: the mux succeeds, then the
// rename over videoPath fails. MergeDASH has already removed videoPath, so it
// must return (mergedPath, err) — the muxed output is the only copy left, and
// vaelor-agent's delivery path recovers it from there. The shim writes the
// output and then turns videoPath into a non-empty directory: os.Remove fails
// (ignored) and os.Rename onto it fails.
func TestMergeDASHRenameFailureReturnsMergedPath(t *testing.T) {
	shimFFmpeg(t, `printf muxed > "$out"; rm -f "$video"; mkdir -p "$video/blocker"`)
	videoPath, audioURL, client := dashFixture(t)

	got, err := media.MergeDASH(context.Background(), client, videoPath, audioURL, 0)
	if err == nil || !strings.Contains(err.Error(), "rename merged file") {
		t.Fatalf("want a rename failure, got %v", err)
	}
	mergedPath := videoPath + ".merged.mp4"
	if got != mergedPath {
		t.Fatalf("returned path = %q, want mergedPath %q (videoPath is gone)", got, mergedPath)
	}
	if b, rerr := os.ReadFile(mergedPath); rerr != nil || string(b) != "muxed" {
		t.Fatalf("muxed output not kept at mergedPath: %q, %v", b, rerr)
	}
	if _, serr := os.Stat(videoPath + ".audio.m4a"); !errors.Is(serr, os.ErrNotExist) {
		t.Errorf("leftover .audio.m4a: stat err=%v, want ErrNotExist", serr)
	}
}

// TestMergeDASHMuxFailureRemovesPartialOutput: ffmpeg dies after writing part
// of the output (a timeout or cancellation kill). MergeDASH returns videoPath,
// so a partial .merged.mp4 left behind would never be found by the caller.
func TestMergeDASHMuxFailureRemovesPartialOutput(t *testing.T) {
	shimFFmpeg(t, `printf partial > "$out"; exit 1`)
	videoPath, audioURL, client := dashFixture(t)

	got, err := media.MergeDASH(context.Background(), client, videoPath, audioURL, 0)
	if err == nil {
		t.Fatal("want a mux failure, got nil")
	}
	if got != videoPath {
		t.Fatalf("returned path = %q, want videoPath %q", got, videoPath)
	}
	if _, serr := os.Stat(videoPath + ".merged.mp4"); !errors.Is(serr, os.ErrNotExist) {
		t.Errorf("partial .merged.mp4 left behind: stat err=%v, want ErrNotExist", serr)
	}
	if b, rerr := os.ReadFile(videoPath); rerr != nil || string(b) != "placeholder" {
		t.Errorf("videoPath not intact after a mux failure: %q, %v", b, rerr)
	}
}
