package youtube

import (
	"context"
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	e := &Extractor{}

	tests := []struct {
		url  string
		want bool
	}{
		// Should match.
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://youtu.be/dQw4w9WgXcQ", true},
		{"https://www.youtube.com/shorts/dQw4w9WgXcQ", true},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", true},
		{"https://music.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://m.youtube.com/watch?v=dQw4w9WgXcQ", true},
		// Variants without www.
		{"https://youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"http://youtu.be/dQw4w9WgXcQ", true},
		// Should NOT match.
		{"https://www.instagram.com/reel/ABC123/", false},
		{"https://vimeo.com/123456", false},
		{"https://example.com/video", false},
		{"not a url", false},
	}

	for _, tt := range tests {
		if got := e.Match(tt.url); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestParseVideoID(t *testing.T) {
	tests := []struct {
		url     string
		wantID  string
		wantErr bool
	}{
		{
			url:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			wantID: "dQw4w9WgXcQ",
		},
		{
			url:    "https://youtu.be/dQw4w9WgXcQ",
			wantID: "dQw4w9WgXcQ",
		},
		{
			url:    "https://www.youtube.com/shorts/dQw4w9WgXcQ",
			wantID: "dQw4w9WgXcQ",
		},
		{
			url:    "https://www.youtube.com/embed/dQw4w9WgXcQ",
			wantID: "dQw4w9WgXcQ",
		},
		{
			url:    "https://music.youtube.com/watch?v=dQw4w9WgXcQ",
			wantID: "dQw4w9WgXcQ",
		},
		{
			url:    "https://m.youtube.com/watch?v=dQw4w9WgXcQ",
			wantID: "dQw4w9WgXcQ",
		},
		{
			url:    "https://youtube.com/watch?v=dQw4w9WgXcQ&t=42",
			wantID: "dQw4w9WgXcQ",
		},
		{
			url:    "https://www.youtube.com/shorts/dQw4w9WgXcQ/",
			wantID: "dQw4w9WgXcQ",
		},
		// Errors.
		{
			url:     "https://www.instagram.com/reel/ABC123/",
			wantErr: true,
		},
		{
			url:     "https://youtube.com/watch",
			wantErr: true,
		},
		{
			url:     "not a url at all",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		id, err := parseVideoID(tt.url)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseVideoID(%q): expected error, got %q", tt.url, id)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseVideoID(%q): unexpected error: %v", tt.url, err)
			continue
		}
		if id != tt.wantID {
			t.Errorf("parseVideoID(%q) = %q, want %q", tt.url, id, tt.wantID)
		}
	}
}

func TestExtractAllBackendsFail(t *testing.T) {
	e := New(Config{YtdlpPath: "/nonexistent/ytdlp"})
	// Use a cancelled context so kkdai fails immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.Extract(ctx, "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err == nil {
		t.Fatal("expected error when all backends fail")
	}
	if !strings.Contains(err.Error(), "all backends failed") {
		t.Errorf("error should mention 'all backends failed', got: %v", err)
	}
}

func TestExtractNoBackends(t *testing.T) {
	e := New(Config{})
	// Use a cancelled context so kkdai fails immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.Extract(ctx, "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	if err == nil {
		t.Fatal("expected error with no working backends")
	}
}
