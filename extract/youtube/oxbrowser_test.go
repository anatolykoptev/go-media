package youtube

import (
	"testing"
	"time"
)

func TestMapToMedia(t *testing.T) {
	dur := 120.0
	views := int64(1000)
	dr := &mediaDownloadResponse{
		MediaType:   "video",
		Files:       []mediaFile{{Path: "/tmp/ox-browser/media/yt_abc123.mp4", SizeBytes: 5000000, Width: 1280, Height: 720}},
		Platform:    "youtube",
		Title:       "Test Video",
		Author:      "Test Author",
		Description: "A test video",
		DurationSec: &dur,
		Stats:       &mediaStats{Views: &views},
		Quality:     &mediaQuality{Width: 1280, Height: 720},
		Merged:      false,
	}

	m := mapToMedia("https://youtube.com/watch?v=test", dr)

	if m.Platform != "youtube" {
		t.Errorf("platform = %q, want youtube", m.Platform)
	}
	if m.Title != "Test Video" {
		t.Errorf("title = %q, want Test Video", m.Title)
	}
	if m.Author != "Test Author" {
		t.Errorf("author = %q, want Test Author", m.Author)
	}
	if m.Description != "A test video" {
		t.Errorf("description = %q, want A test video", m.Description)
	}
	if m.LocalPath != "/tmp/ox-browser/media/yt_abc123.mp4" {
		t.Errorf("localPath = %q, want file path", m.LocalPath)
	}
	if m.Duration != 120*time.Second {
		t.Errorf("duration = %v, want 2m0s", m.Duration)
	}
	if m.Stats.Views != 1000 {
		t.Errorf("views = %d, want 1000", m.Stats.Views)
	}
	if len(m.Qualities) != 1 || m.Qualities[0].Label != "720p" {
		t.Errorf("qualities = %v, want [720p]", m.Qualities)
	}
}

func TestMapToMediaMinimal(t *testing.T) {
	dr := &mediaDownloadResponse{
		MediaType: "video",
		Files:     []mediaFile{{Path: "/tmp/v.mp4", SizeBytes: 100}},
		Platform:  "generic",
	}

	m := mapToMedia("https://example.com/video", dr)

	if m.Platform != "generic" {
		t.Errorf("platform = %q, want generic", m.Platform)
	}
	if m.LocalPath != "/tmp/v.mp4" {
		t.Errorf("localPath = %q", m.LocalPath)
	}
	if m.Duration != 0 {
		t.Errorf("duration = %v, want 0", m.Duration)
	}
	if m.Stats.Views != 0 {
		t.Errorf("views = %d, want 0", m.Stats.Views)
	}
}
