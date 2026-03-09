package media_test

import (
	"context"
	"testing"

	"github.com/anatolykoptev/go-media"
)

type mockExtractor struct {
	name    string
	matches bool
	media   *media.Media
	err     error
}

func (m *mockExtractor) Name() string        { return m.name }
func (m *mockExtractor) Match(_ string) bool { return m.matches }
func (m *mockExtractor) Extract(_ context.Context, url string) (*media.Media, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.media.URL = url
	return m.media, nil
}

func TestRegistryMatch(t *testing.T) {
	r := media.NewRegistry()

	ig := &mockExtractor{name: "instagram", matches: false}
	yt := &mockExtractor{name: "youtube", matches: true}

	r.Register(ig)
	r.Register(yt)

	got := r.Match("https://youtube.com/watch?v=123")
	if got == nil {
		t.Fatal("expected match, got nil")
	}
	if got.Name() != "youtube" {
		t.Fatalf("expected youtube, got %s", got.Name())
	}
}

func TestRegistryMatchNone(t *testing.T) {
	r := media.NewRegistry()
	r.Register(&mockExtractor{name: "ig", matches: false})

	if got := r.Match("https://unknown.com/video"); got != nil {
		t.Fatalf("expected nil, got %s", got.Name())
	}
}

func TestRegistryExtractNoMatch(t *testing.T) {
	r := media.NewRegistry()
	_, err := r.Extract(context.Background(), "https://unknown.com/video")
	if err == nil {
		t.Fatal("expected error for unmatched URL")
	}
}

func TestRegistryExtract(t *testing.T) {
	r := media.NewRegistry()
	r.Register(&mockExtractor{
		name:    "test",
		matches: true,
		media: &media.Media{
			Platform: "test",
			VideoURL: "https://cdn.test.com/video.mp4",
		},
	})

	m, err := r.Extract(context.Background(), "https://test.com/post/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Platform != "test" {
		t.Fatalf("expected platform test, got %s", m.Platform)
	}
	if m.URL != "https://test.com/post/123" {
		t.Fatalf("expected URL to be set, got %s", m.URL)
	}
}

func TestRegistryPlatforms(t *testing.T) {
	r := media.NewRegistry()
	r.Register(&mockExtractor{name: "instagram"})
	r.Register(&mockExtractor{name: "youtube"})

	platforms := r.Platforms()
	if len(platforms) != 2 {
		t.Fatalf("expected 2 platforms, got %d", len(platforms))
	}
	if platforms[0] != "instagram" || platforms[1] != "youtube" {
		t.Fatalf("unexpected platforms: %v", platforms)
	}
}
