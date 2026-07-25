package media_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/anatolykoptev/go-media"
)

// slideMedia builds a *Media with the given slides for the mock extractor.
func slideMedia(slides ...media.Slide) *media.Media {
	return &media.Media{Platform: "test", Slides: slides}
}

// slideServer serves N distinct byte bodies at /slide0, /slide1, ... so each
// slide downloads different content and a test can tell them apart by size.
func slideServer(t *testing.T, n int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /slide{i} returns i+1 bytes so sizes are distinct.
		idx := -1
		_, _ = fmt.Sscanf(r.URL.Path, "/slide%d", &idx)
		if idx < 0 || idx >= n {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(make([]byte, idx+1))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// photoSlides builds N image slides pointing at slideServer's /slide{i}.
func photoSlides(base string, n int) []media.Slide {
	out := make([]media.Slide, n)
	for i := range out {
		out[i] = media.Slide{Type: media.SlideTypeImage, URL: base + "/slide" + strconv.Itoa(i)}
	}
	return out
}

// TestProcessPhotoCarousel: N photo slides in → N ordered image paths out,
// each the downloaded file for its slide. Order must survive into Result.
func TestProcessPhotoCarousel(t *testing.T) {
	const n = 4
	srv := slideServer(t, n)
	tmp := t.TempDir()
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: slideMedia(photoSlides(srv.URL, n)...)}),
		media.WithHTTPClient(srv.Client()),
	)

	res, err := p.Process(context.Background(), "https://test.com/p/1", media.Options{TempDir: tmp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Slides) != n {
		t.Fatalf("len(Slides) = %d, want %d", len(res.Slides), n)
	}
	for i, sr := range res.Slides {
		if sr.Index != i {
			t.Errorf("Slides[%d].Index = %d, want %d (order)", i, sr.Index, i)
		}
		if sr.Type != media.SlideTypeImage {
			t.Errorf("Slides[%d].Type = %d, want image", i, sr.Type)
		}
		if sr.Err != nil {
			t.Errorf("Slides[%d].Err = %v, want nil", i, sr.Err)
		}
		if sr.Path == "" {
			t.Errorf("Slides[%d].Path empty", i)
			continue
		}
		info, err := os.Stat(sr.Path)
		if err != nil {
			t.Errorf("Slides[%d].Path stat: %v", i, err)
			continue
		}
		// slideServer returns i+1 bytes for /slide{i}.
		if got := int(info.Size()); got != i+1 {
			t.Errorf("Slides[%d].Path size = %d, want %d", i, got, i+1)
		}
		if ext := filepath.Ext(sr.Path); ext != ".jpg" {
			t.Errorf("Slides[%d].Path ext = %q, want .jpg", i, ext)
		}
	}
	// Single-video fields must NOT be populated for a carousel.
	if res.VideoPath != "" {
		t.Errorf("VideoPath = %q, want empty for carousel", res.VideoPath)
	}
}

// TestProcessVideoCarousel: every video slide must download. Each slide has a
// DASH audio URL → mergeDASH runs (ffmpeg). When ffmpeg is absent the merge
// fails, which would mark slides as failed — so this test uses self-contained
// video slides (no AudioURL) to assert every slide downloads, and a separate
// test (TestProcessVideoSlideDASHMux) guards the mux path under a ffmpeg skip
// guard. The codec-selection path itself is pinned in
// extract/instagram.TestBuildSlideVideoDASHSelectsH264.
func TestProcessVideoCarousel(t *testing.T) {
	const n = 3
	srv := slideServer(t, n)
	tmp := t.TempDir()
	slides := make([]media.Slide, n)
	for i := range slides {
		slides[i] = media.Slide{Type: media.SlideTypeVideo, URL: srv.URL + "/slide" + strconv.Itoa(i)}
	}
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: slideMedia(slides...)}),
		media.WithHTTPClient(srv.Client()),
	)

	res, err := p.Process(context.Background(), "https://test.com/p/1", media.Options{TempDir: tmp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Slides) != n {
		t.Fatalf("len(Slides) = %d, want %d", len(res.Slides), n)
	}
	for i, sr := range res.Slides {
		if sr.Type != media.SlideTypeVideo {
			t.Errorf("Slides[%d].Type = %d, want video", i, sr.Type)
		}
		if sr.Err != nil {
			t.Errorf("Slides[%d].Err = %v, want nil", i, sr.Err)
		}
		if sr.Path == "" {
			t.Errorf("Slides[%d].Path empty", i)
			continue
		}
		if ext := filepath.Ext(sr.Path); ext != ".mp4" {
			t.Errorf("Slides[%d].Path ext = %q, want .mp4", i, ext)
		}
	}
}

// TestProcessMixedCarouselOrderAndType: a mixed carousel [video, video, photo]
// must preserve order AND per-slide type in the result — the case the old flat
// list could not express.
func TestProcessMixedCarouselOrderAndType(t *testing.T) {
	const n = 3
	srv := slideServer(t, n)
	tmp := t.TempDir()
	slides := []media.Slide{
		{Type: media.SlideTypeVideo, URL: srv.URL + "/slide0"},
		{Type: media.SlideTypeVideo, URL: srv.URL + "/slide1"},
		{Type: media.SlideTypeImage, URL: srv.URL + "/slide2"},
	}
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: slideMedia(slides...)}),
		media.WithHTTPClient(srv.Client()),
	)

	res, err := p.Process(context.Background(), "https://test.com/p/1", media.Options{TempDir: tmp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantTypes := []media.SlideType{media.SlideTypeVideo, media.SlideTypeVideo, media.SlideTypeImage}
	for i, wt := range wantTypes {
		if res.Slides[i].Index != i {
			t.Errorf("Slides[%d].Index = %d, want %d", i, res.Slides[i].Index, i)
		}
		if res.Slides[i].Type != wt {
			t.Errorf("Slides[%d].Type = %d, want %d", i, res.Slides[i].Type, wt)
		}
		if res.Slides[i].Err != nil {
			t.Errorf("Slides[%d].Err = %v, want nil", i, res.Slides[i].Err)
		}
	}
}

// TestProcessSinglePhotoNoVideoURL: a single photo post must NOT hit the
// "no video URL found" guard — it downloads the one photo slide.
func TestProcessSinglePhotoNoVideoURL(t *testing.T) {
	srv := slideServer(t, 1)
	tmp := t.TempDir()
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: slideMedia(
			media.Slide{Type: media.SlideTypeImage, URL: srv.URL + "/slide0"},
		)}),
		media.WithHTTPClient(srv.Client()),
	)

	res, err := p.Process(context.Background(), "https://test.com/p/1", media.Options{TempDir: tmp})
	if err != nil {
		t.Fatalf("unexpected error (photo must not hit no-video-URL guard): %v", err)
	}
	if len(res.Slides) != 1 || res.Slides[0].Path == "" {
		t.Fatalf("expected 1 downloaded photo slide, got %+v", res.Slides)
	}
}

// TestProcessSingleVideoRegression: a single video (no Slides) must behave
// byte-for-byte as today — VideoPath set, Slides empty, transcription nil when
// no transcriber. This pins the single-video path against the carousel branch.
func TestProcessSingleVideoRegression(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("fake-video-content"))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: &media.Media{
			Platform: "test", VideoURL: srv.URL + "/video.mp4",
		}}),
		media.WithHTTPClient(srv.Client()),
	)

	res, err := p.Process(context.Background(), "https://test.com/post/1", media.Options{TempDir: tmp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Single-video path: VideoPath set, Slides empty.
	if res.VideoPath == "" {
		t.Fatal("VideoPath empty, want the single-video download path")
	}
	if len(res.Slides) != 0 {
		t.Fatalf("len(Slides) = %d, want 0 (single video must not divert to carousel)", len(res.Slides))
	}
	if res.Transcription != nil {
		t.Fatal("Transcription non-nil, want nil with no transcriber")
	}
	info, err := os.Stat(res.VideoPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("video file missing/empty: %v", err)
	}
}

// TestProcessPartialFailure: when one slide's download fails, the caller must
// detect it — Process returns a non-nil error (a *SlideError) together with a
// Result whose Slides slice reports expected count and which slides failed.
// The successful slides must still carry their paths.
func TestProcessPartialFailure(t *testing.T) {
	// Serve slides 0 and 2; slide 1 returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slide1" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	tmp := t.TempDir()
	slides := []media.Slide{
		{Type: media.SlideTypeImage, URL: srv.URL + "/slide0"},
		{Type: media.SlideTypeImage, URL: srv.URL + "/slide1"}, // fails
		{Type: media.SlideTypeImage, URL: srv.URL + "/slide2"},
	}
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: slideMedia(slides...)}),
		media.WithHTTPClient(srv.Client()),
	)

	res, err := p.Process(context.Background(), "https://test.com/p/1", media.Options{TempDir: tmp})
	if err == nil {
		t.Fatal("expected error for partial slide failure, got nil")
	}
	var se *media.SlideError
	if !errors.As(err, &se) {
		t.Fatalf("error = %T, want *media.SlideError", err)
	}
	if se.Expected != 3 || se.Succeeded != 2 || len(se.Failed) != 1 || se.Failed[0] != 1 {
		t.Fatalf("SlideError = %+v, want Expected=3 Succeeded=2 Failed=[1]", se)
	}
	// The Result must still be returned with the full slide detail.
	if res == nil {
		t.Fatal("Result nil, want partial result")
	}
	if len(res.Slides) != 3 {
		t.Fatalf("len(res.Slides) = %d, want 3 (expected count)", len(res.Slides))
	}
	// Slides 0 and 2 succeed (Path set, no Err); slide 1 fails (no Path, Err set).
	wantOK := []int{0, 2}
	for _, i := range wantOK {
		if res.Slides[i].Path == "" || res.Slides[i].Err != nil {
			t.Errorf("slide %d: Path=%q Err=%v, want success", i, res.Slides[i].Path, res.Slides[i].Err)
		}
	}
	if res.Slides[1].Path != "" || res.Slides[1].Err == nil {
		t.Errorf("slide 1: Path=%q Err=%v, want failure", res.Slides[1].Path, res.Slides[1].Err)
	}
}

// TestProcessTextOnlyNoPanic: a text-only post (no slides, no VideoURL) must
// return an error, not panic.
func TestProcessTextOnlyNoPanic(t *testing.T) {
	tmp := t.TempDir()
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "test", matches: true, media: &media.Media{Platform: "test"}}),
	)
	_, err := p.Process(context.Background(), "https://test.com/p/1", media.Options{TempDir: tmp})
	if err == nil {
		t.Fatal("expected error for text-only post, got nil")
	}
}
