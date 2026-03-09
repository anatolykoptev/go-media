package youtube

import (
	"strings"
	"testing"
	"time"
)

const samplePlayerHTML = `<html><head></head><body>
<script>var ytInitialPlayerResponse = {"videoDetails":{"title":"Test Video","shortDescription":"A test description","lengthSeconds":"120"},"streamingData":{"formats":[{"itag":18,"url":"https://example.com/video.mp4","mimeType":"video/mp4","width":640,"height":360,"bitrate":500000}],"adaptiveFormats":[{"itag":137,"url":"https://example.com/hd.mp4","mimeType":"video/mp4","width":1920,"height":1080,"bitrate":4000000},{"itag":140,"url":"https://example.com/audio.m4a","mimeType":"audio/mp4","width":0,"height":0,"bitrate":128000}]}};</script>
</body></html>`

func TestParsePlayerResponse(t *testing.T) {
	pr, err := parsePlayerResponse(samplePlayerHTML)
	if err != nil {
		t.Fatalf("parsePlayerResponse: %v", err)
	}
	if pr.VideoDetails.Title != "Test Video" {
		t.Errorf("title = %q, want %q", pr.VideoDetails.Title, "Test Video")
	}
	if pr.VideoDetails.Description != "A test description" {
		t.Errorf("description = %q, want %q", pr.VideoDetails.Description, "A test description")
	}
	if pr.VideoDetails.LengthSec != "120" {
		t.Errorf("lengthSeconds = %q, want %q", pr.VideoDetails.LengthSec, "120")
	}
	if len(pr.StreamingData.Formats) != 1 {
		t.Errorf("formats count = %d, want 1", len(pr.StreamingData.Formats))
	}
	if len(pr.StreamingData.AdaptiveFormats) != 2 {
		t.Errorf("adaptive formats count = %d, want 2", len(pr.StreamingData.AdaptiveFormats))
	}
}

func TestParsePlayerResponseNotFound(t *testing.T) {
	cases := []struct {
		name string
		html string
	}{
		{"empty", ""},
		{"no script", "<html><body>Hello</body></html>"},
		{"wrong var", `<script>var ytOtherData = {};</script>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parsePlayerResponse(tc.html)
			if err == nil {
				t.Fatal("expected error for missing player response")
			}
			if !strings.Contains(err.Error(), "player response not found") {
				t.Errorf("error = %v, want 'player response not found'", err)
			}
		})
	}
}

func TestBuildMedia(t *testing.T) {
	pr := &playerResponse{}
	pr.VideoDetails.Title = "My Video"
	pr.VideoDetails.Description = "My Description"
	pr.VideoDetails.LengthSec = "90"
	pr.StreamingData.Formats = []playerFormat{
		{ITag: 18, URL: "https://example.com/360.mp4", MimeType: "video/mp4", Width: 640, Height: 360, Bitrate: 500000},
		{ITag: 22, URL: "https://example.com/720.mp4", MimeType: "video/mp4", Width: 1280, Height: 720, Bitrate: 2000000},
	}

	m, err := buildMedia("https://www.youtube.com/watch?v=test123", pr)
	if err != nil {
		t.Fatalf("buildMedia: %v", err)
	}
	if m.Platform != "youtube" {
		t.Errorf("platform = %q, want %q", m.Platform, "youtube")
	}
	if m.Title != "My Video" {
		t.Errorf("title = %q, want %q", m.Title, "My Video")
	}
	if m.Description != "My Description" {
		t.Errorf("description = %q, want %q", m.Description, "My Description")
	}
	if m.Duration != 90*time.Second {
		t.Errorf("duration = %v, want %v", m.Duration, 90*time.Second)
	}
	// Should pick 720p (highest).
	if m.VideoURL != "https://example.com/720.mp4" {
		t.Errorf("videoURL = %q, want 720p URL", m.VideoURL)
	}
	if len(m.Qualities) != 2 {
		t.Errorf("qualities count = %d, want 2", len(m.Qualities))
	}
}

func TestBuildMediaAdaptiveOnly(t *testing.T) {
	pr := &playerResponse{}
	pr.VideoDetails.Title = "Adaptive"
	pr.StreamingData.AdaptiveFormats = []playerFormat{
		{ITag: 137, URL: "https://example.com/video.mp4", MimeType: "video/mp4", Width: 1920, Height: 1080, Bitrate: 4000000},
		{ITag: 140, URL: "https://example.com/audio.m4a", MimeType: "audio/mp4", Bitrate: 128000},
	}

	m, err := buildMedia("https://youtu.be/test123", pr)
	if err != nil {
		t.Fatalf("buildMedia: %v", err)
	}
	if m.VideoURL != "https://example.com/video.mp4" {
		t.Errorf("videoURL = %q, want video adaptive URL", m.VideoURL)
	}
	if m.AudioURL != "https://example.com/audio.m4a" {
		t.Errorf("audioURL = %q, want audio adaptive URL", m.AudioURL)
	}
}

func TestBuildMediaNoDirectURLs(t *testing.T) {
	pr := &playerResponse{}
	pr.VideoDetails.Title = "Cipher Only"
	pr.StreamingData.Formats = []playerFormat{
		{ITag: 18, URL: "", MimeType: "video/mp4", Width: 640, Height: 360},
	}
	pr.StreamingData.AdaptiveFormats = []playerFormat{
		{ITag: 137, URL: "", MimeType: "video/mp4", Width: 1920, Height: 1080},
	}

	_, err := buildMedia("https://youtu.be/test123", pr)
	if err == nil {
		t.Fatal("expected error for no direct URLs")
	}
	if !strings.Contains(err.Error(), "no direct URLs available") {
		t.Errorf("error = %v, want 'no direct URLs available'", err)
	}
}
