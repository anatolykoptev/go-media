package media_test

import (
	"context"
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
