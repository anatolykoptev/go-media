package instagram

import (
	"testing"

	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
)

// fallbackVideoURL is the video_versions URL used across the fallback tests.
const fallbackVideoURL = "https://cdn.instagram.com/720.mp4"

// dashManifestFixture mirrors the structure tested in extract/dash: four video
// representations 240p→1080p, one audio, 30s duration, absolute BaseURLs.
const dashManifestFixture = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT0H0M30.000S" type="static">
  <Period>
    <AdaptationSet mimeType="video/mp4" contentType="video">
      <Representation id="240p" width="426" height="240" bandwidth="80000" codecs="avc1.4d401e">
        <BaseURL>https://video-edge-240p.example.com/video.mp4</BaseURL>
      </Representation>
      <Representation id="480p" width="854" height="480" bandwidth="400000" codecs="avc1.4d401f">
        <BaseURL>https://video-edge-480p.example.com/video.mp4</BaseURL>
      </Representation>
      <Representation id="720p" width="1280" height="720" bandwidth="1100000" codecs="avc1.4d401f">
        <BaseURL>https://video-edge-720p.example.com/video.mp4</BaseURL>
      </Representation>
      <Representation id="1080p" width="1080" height="1920" bandwidth="2301000" codecs="avc1.640028">
        <BaseURL>https://video-edge-1080p.example.com/video.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet mimeType="audio/mp4" contentType="audio">
      <Representation id="audio" bandwidth="64000" codecs="mp4a.40.2">
        <BaseURL>https://video-edge-audio.example.com/audio.m4a</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

func TestMatch(t *testing.T) {
	e := &Extractor{}

	tests := []struct {
		url  string
		want bool
	}{
		{"https://www.instagram.com/reel/ABC123/", true},
		{"https://instagram.com/p/XYZ789/", true},
		{"https://www.threads.net/@user/post/ABC123", true},
		{"https://youtube.com/watch?v=123", false},
		{"https://example.com/video", false},
		{"not a url", false},
	}

	for _, tt := range tests {
		if got := e.Match(tt.url); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		url         string
		igCode      string
		threadsUser string
		threadsCode string
		wantErr     bool
	}{
		{
			url:    "https://www.instagram.com/reel/ABC123/",
			igCode: "ABC123",
		},
		{
			url:    "https://instagram.com/p/XYZ-789_abc/",
			igCode: "XYZ-789_abc",
		},
		{
			url:         "https://www.threads.net/@johndoe/post/DEF456",
			threadsUser: "johndoe",
			threadsCode: "DEF456",
		},
		{
			url:     "https://youtube.com/watch?v=123",
			wantErr: true,
		},
		{
			url:     "not a url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		ig, tu, tc, err := parseURL(tt.url)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseURL(%q): expected error, got none", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseURL(%q): unexpected error: %v", tt.url, err)
			continue
		}
		if ig != tt.igCode {
			t.Errorf("parseURL(%q): igCode = %q, want %q", tt.url, ig, tt.igCode)
		}
		if tu != tt.threadsUser {
			t.Errorf("parseURL(%q): threadsUser = %q, want %q", tt.url, tu, tt.threadsUser)
		}
		if tc != tt.threadsCode {
			t.Errorf("parseURL(%q): threadsCode = %q, want %q", tt.url, tc, tt.threadsCode)
		}
	}
}

func TestMapStats(t *testing.T) {
	t.Run("all metrics", func(t *testing.T) {
		post := threads.Post{
			ViewCount:    56964765,
			LikeCount:    2244564,
			CommentCount: 24885,
			RepostCount:  27745,
			ReplyCount:   999, // ignored when CommentCount set
		}
		got := mapStats(post)
		want := media.MediaStats{
			Views:    56964765,
			Likes:    2244564,
			Comments: 24885,
			Shares:   27745,
		}
		if got != want {
			t.Fatalf("mapStats = %+v, want %+v", got, want)
		}
	})

	t.Run("comment fallback to reply count", func(t *testing.T) {
		// Threads posts expose direct_reply_count but no IG comment_count.
		post := threads.Post{
			LikeCount:    100,
			ReplyCount:   5,
			CommentCount: 0,
		}
		got := mapStats(post)
		if got.Comments != 5 {
			t.Fatalf("Comments = %d, want 5 (fallback from ReplyCount)", got.Comments)
		}
		if got.Likes != 100 {
			t.Errorf("Likes = %d, want 100", got.Likes)
		}
	})
}

func TestPopulateMediaWithManifest(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{
		Videos: []threads.MediaVersion{
			{URL: fallbackVideoURL, Width: 720, Height: 1280},
		},
		VideoDashManifest: dashManifestFixture,
	}

	populateMedia(m, post, 0)

	// DASH pair must be set: 1080p video (best, no budget) + audio.
	if m.VideoURL != "https://video-edge-1080p.example.com/video.mp4" {
		t.Fatalf("VideoURL = %q, want 1080p DASH url", m.VideoURL)
	}
	if m.AudioURL != "https://video-edge-audio.example.com/audio.m4a" {
		t.Fatalf("AudioURL = %q, want DASH audio url", m.AudioURL)
	}
	// Qualities populated from all video representations.
	if len(m.Qualities) != 4 {
		t.Fatalf("Qualities: got %d, want 4", len(m.Qualities))
	}
	// The 1080p quality must carry dimensions and an estimated size.
	var top *media.Quality
	for i := range m.Qualities {
		if m.Qualities[i].Height == 1920 {
			top = &m.Qualities[i]
		}
	}
	if top == nil {
		t.Fatal("no 1080p (height=1920) quality found")
	}
	if top.Width != 1080 || top.Label != "1080p" {
		t.Fatalf("1080p quality = %+v", top)
	}
	if top.Size != 8628750 { // 2301000 * 30 / 8
		t.Fatalf("1080p Size = %d, want 8628750", top.Size)
	}
}

func TestPopulateMediaManifestBudgetPicksLower(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{
		Videos: []threads.MediaVersion{
			{URL: fallbackVideoURL, Width: 720, Height: 1280},
		},
		VideoDashManifest: dashManifestFixture,
	}
	// Budget that fits 720p (4125000) but not 1080p (8628750).
	populateMedia(m, post, 4125000)
	if m.VideoURL != "https://video-edge-720p.example.com/video.mp4" {
		t.Fatalf("VideoURL = %q, want 720p (budget-capped)", m.VideoURL)
	}
	if m.AudioURL == "" {
		t.Fatal("AudioURL empty, want DASH audio url")
	}
}

func TestPopulateMediaNoManifestFallsBackToVideoVersions(t *testing.T) {
	// Regression guard for the embed/SSR/proxy rungs that never carry a
	// manifest: behaviour must be exactly today's video_versions path.
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{
		Videos: []threads.MediaVersion{
			{URL: fallbackVideoURL, Width: 720, Height: 1280},
			{URL: "https://cdn.instagram.com/480.mp4", Width: 480, Height: 854},
		},
		VideoDashManifest: "", // no manifest — fallback rung
	}

	populateMedia(m, post, 50_000_000)

	if m.VideoURL != fallbackVideoURL {
		t.Fatalf("VideoURL = %q, want first video_versions entry", m.VideoURL)
	}
	if m.AudioURL != "" {
		t.Fatalf("AudioURL = %q, want empty (no DASH on fallback rung)", m.AudioURL)
	}
	if len(m.Qualities) != 2 {
		t.Fatalf("Qualities: got %d, want 2 (video_versions)", len(m.Qualities))
	}
}

func TestPopulateMediaUnparseableManifestFallsBack(t *testing.T) {
	// A malformed manifest must NOT error out of Extract — it must fall back
	// to video_versions so the embed/SSR/proxy rungs keep working.
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{
		Videos: []threads.MediaVersion{
			{URL: fallbackVideoURL, Width: 720, Height: 1280},
		},
		VideoDashManifest: "<?xml broken",
	}

	populateMedia(m, post, 0)
	if m.VideoURL != fallbackVideoURL {
		t.Fatalf("VideoURL = %q, want video_versions fallback", m.VideoURL)
	}
	if m.AudioURL != "" {
		t.Fatalf("AudioURL = %q, want empty on fallback", m.AudioURL)
	}
}

// noBaseURLManifestFixture is a manifest whose representations carry NO
// BaseURL — exercises the defensive guard: populateMedia must fall back to
// video_versions instead of committing to empty-URL DASH reps that would
// hard-fail the download.
const noBaseURLManifestFixture = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT0H0M30.000S" type="static">
  <Period>
    <AdaptationSet mimeType="video/mp4" contentType="video">
      <Representation id="240p" width="426" height="240" bandwidth="80000" codecs="avc1.4d401e"/>
      <Representation id="1080p" width="1080" height="1920" bandwidth="2301000" codecs="avc1.640028"/>
    </AdaptationSet>
    <AdaptationSet mimeType="audio/mp4" contentType="audio">
      <Representation id="audio" bandwidth="64000" codecs="mp4a.40.2"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestPopulateMediaManifestNoBaseURLFallsBack(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{
		Videos: []threads.MediaVersion{
			{URL: fallbackVideoURL, Width: 720, Height: 1280},
		},
		VideoDashManifest: noBaseURLManifestFixture,
	}

	populateMedia(m, post, 0)

	if m.VideoURL != fallbackVideoURL {
		t.Fatalf("VideoURL = %q, want video_versions fallback (no BaseURL in manifest)", m.VideoURL)
	}
	if m.AudioURL != "" {
		t.Fatalf("AudioURL = %q, want empty on fallback", m.AudioURL)
	}
}
