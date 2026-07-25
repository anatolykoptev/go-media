package dash

import (
	"testing"
)

// fixtureMPD is a trimmed real-shaped Instagram DASH manifest: multiple video
// representations 240p→1080p, one audio representation, BaseURL elements, and a
// 30s mediaPresentationDuration. Bandwidths chosen so size estimates are
// round: size = bandwidth(bps) * duration(s) / 8.
const fixtureMPD = `<?xml version="1.0" encoding="UTF-8"?>
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

// estimatedSize: bandwidth(bps) * fixtureDuration(s) / 8.
const fixtureDuration = 30.0

func estimatedSize(bandwidth int64) int64 {
	return int64(float64(bandwidth) * fixtureDuration / 8)
}

func TestParseManifest(t *testing.T) {
	man, err := ParseManifest(fixtureMPD)
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}

	if man.Duration != 30.0 {
		t.Fatalf("Duration = %v, want 30.0", man.Duration)
	}

	if len(man.Videos) != 4 {
		t.Fatalf("Videos: got %d, want 4", len(man.Videos))
	}
	if len(man.Audios) != 1 {
		t.Fatalf("Audios: got %d, want 1", len(man.Audios))
	}

	// Spot-check the 1080p representation (highest).
	r := man.Videos[3]
	if r.ID != "1080p" || r.Width != 1080 || r.Height != 1920 || r.Bandwidth != 2301000 {
		t.Fatalf("1080p rep = %+v", r)
	}
	if r.MimeType != "video/mp4" {
		t.Fatalf("1080p MimeType = %q, want video/mp4", r.MimeType)
	}
	if r.Codecs != "avc1.640028" {
		t.Fatalf("1080p Codecs = %q, want avc1.640028", r.Codecs)
	}
	if r.URL != "https://video-edge-1080p.example.com/video.mp4" {
		t.Fatalf("1080p URL = %q", r.URL)
	}

	// Audio representation.
	a := man.Audios[0]
	if a.Bandwidth != 64000 || a.URL != "https://video-edge-audio.example.com/audio.m4a" {
		t.Fatalf("audio rep = %+v", a)
	}
	if a.MimeType != "audio/mp4" {
		t.Fatalf("audio MimeType = %q, want audio/mp4", a.MimeType)
	}
}

func TestParseManifestMalformed(t *testing.T) {
	// Truncated/unbalanced XML must return an error, never panic.
	_, err := ParseManifest(`<?xml version="1.0"?><MPD><Period><AdaptationSet>`)
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}

	// Empty string must error, not panic.
	if _, err := ParseManifest(""); err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestParseManifestNoVideo(t *testing.T) {
	// An MPD with only an audio AdaptationSet is unusable for the mux path.
	const onlyAudio = `<?xml version="1.0"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT0H0M10.000S">
  <Period>
    <AdaptationSet mimeType="audio/mp4">
      <Representation id="audio" bandwidth="64000" codecs="mp4a.40.2">
        <BaseURL>https://example.com/audio.m4a</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`
	man, err := ParseManifest(onlyAudio)
	if err != nil {
		t.Fatalf("ParseManifest: unexpected error: %v", err)
	}
	if len(man.Videos) != 0 {
		t.Fatalf("Videos: got %d, want 0", len(man.Videos))
	}
}

func TestSelect(t *testing.T) {
	man, err := ParseManifest(fixtureMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}

	tests := []struct {
		name        string
		budget      int64
		wantHeight  int
		wantAudioBW int64
	}{
		{
			name:        "no budget picks best (1080p)",
			budget:      0,
			wantHeight:  1920,
			wantAudioBW: 64000,
		},
		{
			name:        "budget admits 1080p",
			budget:      estimatedSize(2301000) + 1, // 8628751
			wantHeight:  1920,
			wantAudioBW: 64000,
		},
		{
			name:        "budget admits 720p not 1080p",
			budget:      estimatedSize(1100000), // 4125000
			wantHeight:  720,
			wantAudioBW: 64000,
		},
		{
			name:        "budget admits only 480p",
			budget:      estimatedSize(400000), // 1500000
			wantHeight:  480,
			wantAudioBW: 64000,
		},
		{
			name:        "budget smaller than everything degrades to smallest",
			budget:      1,
			wantHeight:  240,
			wantAudioBW: 64000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video, audio, err := Select(man, tt.budget)
			if err != nil {
				t.Fatalf("Select: unexpected error: %v", err)
			}
			if video.Height != tt.wantHeight {
				t.Fatalf("video height = %d, want %d", video.Height, tt.wantHeight)
			}
			if audio.Bandwidth != tt.wantAudioBW {
				t.Fatalf("audio bandwidth = %d, want %d", audio.Bandwidth, tt.wantAudioBW)
			}
		})
	}
}

func TestSelectNeverPicksLargerThanBudget(t *testing.T) {
	man, err := ParseManifest(fixtureMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	// Budget exactly fits 480p. 720p must never be chosen.
	budget := estimatedSize(400000)
	video, _, err := Select(man, budget)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if video.Height > 480 {
		t.Fatalf("picked height %d over budget that caps at 480p", video.Height)
	}
}

func TestSelectNoVideoError(t *testing.T) {
	man := &Manifest{Duration: 10, Audios: []Representation{{Bandwidth: 64000, URL: "x"}}}
	_, _, err := Select(man, 0)
	if err == nil {
		t.Fatal("expected error when no video representations, got nil")
	}
}

func TestPickVideoNoURLReturnsZero(t *testing.T) {
	// No representation carries a BaseURL: pickVideo must signal "no usable
	// video" (zero Representation) so the caller degrades instead of
	// returning an empty-URL rep that hard-fails the download.
	reps := []Representation{
		{ID: "a", Height: 720, Bandwidth: 1000, URL: ""},
		{ID: "b", Height: 1080, Bandwidth: 2000, URL: ""},
	}
	got := pickVideo(reps, 30, 0)
	if got != (Representation{}) {
		t.Fatalf("pickVideo no-URL reps = %+v, want zero Representation (signal degrade)", got)
	}
}

func TestPickAudioNoURLReturnsZero(t *testing.T) {
	reps := []Representation{
		{ID: "a", Bandwidth: 64000, URL: ""},
	}
	got := pickAudio(reps)
	if got != (Representation{}) {
		t.Fatalf("pickAudio no-URL reps = %+v, want zero Representation (signal degrade)", got)
	}
}

func TestSelectZeroDurationBudgetPicksSmallest(t *testing.T) {
	// Missing duration → EstimatedSize returns 0 for every rep, so the old
	// "fits" check treated everything as fitting and picked the HIGHEST
	// resolution, silently uncapping the budget. With a budget set and
	// unknown duration the selector must pick the SMALLEST (safe default).
	man, err := ParseManifest(fixtureMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	man.Duration = 0
	video, _, err := Select(man, 1)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if video.Height != 240 {
		t.Fatalf("zero-duration + budget: picked height %d, want 240 (smallest)", video.Height)
	}
}

func TestParseManifestQualityRankingNotLabel(t *testing.T) {
	// qualityRanking is a NUMERIC rank (1 = best), not a display label. It
	// must not be rendered as Representation.Label.
	const mpd = `<?xml version="1.0"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT0H0M10.000S">
  <Period>
    <AdaptationSet mimeType="video/mp4">
      <Representation id="v0" width="1280" height="720" bandwidth="1100000" qualityRanking="1"/>
    </AdaptationSet>
  </Period>
</MPD>`
	man, err := ParseManifest(mpd)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(man.Videos) != 1 {
		t.Fatalf("Videos: got %d, want 1", len(man.Videos))
	}
	if man.Videos[0].Label != "" {
		t.Fatalf("Label = %q, want empty (qualityRanking is a rank, not a label)", man.Videos[0].Label)
	}
}
