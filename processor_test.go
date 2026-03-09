package media_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/anatolykoptev/go-media"
)

func TestNewProcessorPlatforms(t *testing.T) {
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{name: "instagram"}),
		media.WithExtractor(&mockExtractor{name: "youtube"}),
	)

	platforms := p.Platforms()
	if len(platforms) != 2 {
		t.Fatalf("expected 2 platforms, got %d", len(platforms))
	}
}

func TestProcessorExtractNoMatch(t *testing.T) {
	p := media.NewProcessor()
	_, err := p.Extract(context.Background(), "https://unknown.com/video")
	if err == nil {
		t.Fatal("expected error for unmatched URL")
	}
}

func TestProcessorExtract(t *testing.T) {
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{
			name:    "test",
			matches: true,
			media: &media.Media{
				Platform: "test",
				VideoURL: "https://cdn.test.com/v.mp4",
			},
		}),
	)

	m, err := p.Extract(context.Background(), "https://test.com/post/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.VideoURL != "https://cdn.test.com/v.mp4" {
		t.Fatalf("unexpected video URL: %s", m.VideoURL)
	}
}

func TestProcessorProcessNoVideoURL(t *testing.T) {
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{
			name:    "test",
			matches: true,
			media:   &media.Media{Platform: "test"}, // no VideoURL
		}),
	)

	_, err := p.Process(context.Background(), "https://test.com/post/1", media.Options{})
	if err == nil {
		t.Fatal("expected error for missing video URL")
	}
}

func TestProcessorProcessSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("fake-video-content"))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{
			name:    "test",
			matches: true,
			media:   &media.Media{Platform: "test", VideoURL: srv.URL + "/video.mp4"},
		}),
		media.WithHTTPClient(srv.Client()),
	)

	result, err := p.Process(context.Background(), "https://test.com/post/1", media.Options{TempDir: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Media.Platform != "test" {
		t.Fatalf("expected platform test, got %s", result.Media.Platform)
	}

	info, err := os.Stat(result.VideoPath)
	if err != nil {
		t.Fatalf("video file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("video file is empty")
	}
}

func TestProcessorProcessDownloadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{
			name:    "test",
			matches: true,
			media:   &media.Media{Platform: "test", VideoURL: srv.URL + "/video.mp4"},
		}),
		media.WithHTTPClient(srv.Client()),
	)

	_, err := p.Process(context.Background(), "https://test.com/post/1", media.Options{TempDir: tmpDir})
	if err == nil {
		t.Fatal("expected error for failed download")
	}
	if !strings.Contains(err.Error(), "download") {
		t.Fatalf("expected error to contain 'download', got: %v", err)
	}
}

func TestProcessorProcessNoTranscriber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write([]byte("fake-video-content"))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	p := media.NewProcessor(
		media.WithExtractor(&mockExtractor{
			name:    "test",
			matches: true,
			media:   &media.Media{Platform: "test", VideoURL: srv.URL + "/video.mp4"},
		}),
		media.WithHTTPClient(srv.Client()),
		// no WithTranscriber — transcription should be nil
	)

	result, err := p.Process(context.Background(), "https://test.com/post/1", media.Options{TempDir: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Transcription != nil {
		t.Fatal("expected nil transcription when no transcriber configured")
	}
}
