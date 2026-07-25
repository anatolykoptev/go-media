package dash

import "testing"

// vp9OnlyMPD mirrors the measured production case (reel DO8cvGViIPu): every
// video representation is VP9 (codecs="vp09..."), audio is HE-AAC
// (mp4a.40.5). Instagram's DASH manifest content varies per upload — some
// posts carry ONLY VP9 video reps, and Telegram's mobile clients cannot decode
// a VP9 video track (blank picture, working audio). The selector must signal
// "no usable video" for such a manifest so the extractor falls through to the
// H.264 video_versions rendition.
const vp9OnlyMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT47.5S" type="static">
  <Period>
    <AdaptationSet contentType="video">
      <Representation id="1447419429847706v" bandwidth="2300928" codecs="vp09.00.40.08.00.01.01.01.00" mimeType="video/mp4" sar="1:1" FBContentLength="13661765" width="1080" height="1920" frameRate="15360/256">
        <BaseURL>https://cdn.example.invalid/vp9-1080p.mp4</BaseURL>
      </Representation>
      <Representation id="240pv" bandwidth="166515" codecs="vp09.00.31.08.00.01.01.01.00" mimeType="video/mp4" FBContentLength="988686" width="720" height="1280">
        <BaseURL>https://cdn.example.invalid/vp9-240p.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet contentType="audio">
      <Representation id="audio" bandwidth="64000" codecs="mp4a.40.5" mimeType="audio/mp4" FBContentLength="359339">
        <BaseURL>https://cdn.example.invalid/audio.m4a</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

// mixedMPD carries both an H.264 and a VP9 video representation. The VP9 rep
// is higher resolution AND fits the budget — a codec-agnostic selector would
// pick it. The H.264 rep must win because Telegram cannot decode VP9.
const mixedMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT30S" type="static">
  <Period>
    <AdaptationSet contentType="video">
      <Representation id="h264-720" width="1280" height="720" bandwidth="1100000" codecs="avc1.4d401f" mimeType="video/mp4" FBContentLength="4125000">
        <BaseURL>https://cdn.example.invalid/h264-720.mp4</BaseURL>
      </Representation>
      <Representation id="vp9-1080" width="1080" height="1920" bandwidth="2300928" codecs="vp09.00.40.08" mimeType="video/mp4" FBContentLength="13661765">
        <BaseURL>https://cdn.example.invalid/vp9-1080.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet contentType="audio">
      <Representation id="audio" bandwidth="64000" codecs="mp4a.40.5" mimeType="audio/mp4">
        <BaseURL>https://cdn.example.invalid/audio.m4a</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

// h264RungsMPD has several H.264 rungs plus one VP9 rung that is the smallest.
// Budget interaction: among H.264 rungs the budget rules still pick the right
// one; when none fits, the smallest H.264 rung is picked — never the VP9 one.
const h264RungsMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT30S" type="static">
  <Period>
    <AdaptationSet contentType="video">
      <Representation id="h264-240" width="426" height="240" bandwidth="80000" codecs="avc1.4d401e" mimeType="video/mp4" FBContentLength="300000">
        <BaseURL>https://cdn.example.invalid/h264-240.mp4</BaseURL>
      </Representation>
      <Representation id="h264-480" width="854" height="480" bandwidth="400000" codecs="avc1.4d401f" mimeType="video/mp4" FBContentLength="1500000">
        <BaseURL>https://cdn.example.invalid/h264-480.mp4</BaseURL>
      </Representation>
      <Representation id="h264-720" width="1280" height="720" bandwidth="1100000" codecs="avc1.4d401f" mimeType="video/mp4" FBContentLength="4125000">
        <BaseURL>https://cdn.example.invalid/h264-720.mp4</BaseURL>
      </Representation>
      <Representation id="vp9-180" width="320" height="180" bandwidth="50000" codecs="vp09.00.30.08" mimeType="video/mp4" FBContentLength="187500">
        <BaseURL>https://cdn.example.invalid/vp9-180.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet contentType="audio">
      <Representation id="audio" bandwidth="64000" codecs="mp4a.40.2" mimeType="audio/mp4">
        <BaseURL>https://cdn.example.invalid/audio.m4a</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

func TestSelectVP9OnlyReportsNoUsableVideo(t *testing.T) {
	man, err := ParseManifest(vp9OnlyMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(man.Videos) != 2 {
		t.Fatalf("Videos: got %d, want 2", len(man.Videos))
	}
	video, _, err := Select(man, 50*1024*1024)
	if err != nil {
		t.Fatalf("Select: unexpected error: %v (VP9-only must signal no usable video via zero rep, not error)", err)
	}
	// No H.264 representation exists → pickVideo must return a zero
	// Representation (empty URL) so the caller degrades to video_versions.
	if video != (Representation{}) {
		t.Fatalf("VP9-only manifest: Select returned %+v, want zero Representation (no H.264 → signal degrade)", video)
	}
}

func TestSelectMixedManifestPicksH264(t *testing.T) {
	man, err := ParseManifest(mixedMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	// Budget admits both reps. The VP9 rep is higher resolution (1080x1920)
	// and would win a codec-agnostic sort. The H.264 rep (720p) must win.
	video, _, err := Select(man, 50*1024*1024)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if video.Codecs == "" || !isH264Codec(video.Codecs) {
		t.Fatalf("mixed manifest: picked non-H.264 rep %+v (codecs=%q), want H.264", video, video.Codecs)
	}
	if video.Height != 720 {
		t.Fatalf("mixed manifest: picked height %d, want 720 (the only H.264 rep)", video.Height)
	}
}

func TestSelectBudgetAmongH264IgnoresVP9(t *testing.T) {
	man, err := ParseManifest(h264RungsMPD)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	// Budget fits 480p (1500000) but not 720p (4125000): pick 480p H.264.
	video, _, err := Select(man, 1500000)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if video.Height != 480 {
		t.Fatalf("budget admits 480p: picked height %d, want 480", video.Height)
	}
	if !isH264Codec(video.Codecs) {
		t.Fatalf("budget admits 480p: picked non-H.264 %q", video.Codecs)
	}

	// Budget smaller than every H.264 rung: degrade to the smallest H.264
	// (240p, 300000), NEVER the VP9 180p rung (187500) even though it is the
	// absolute smallest and would be picked by a codec-agnostic selector.
	videoSmall, _, err := Select(man, 1)
	if err != nil {
		t.Fatalf("Select(smallest): %v", err)
	}
	if videoSmall.Height != 240 {
		t.Fatalf("budget below all H.264: picked height %d, want 240 (smallest H.264), not the VP9 180p rung", videoSmall.Height)
	}
	if !isH264Codec(videoSmall.Codecs) {
		t.Fatalf("budget below all H.264: picked non-H.264 %q — VP9 must never be a degrade target", videoSmall.Codecs)
	}
}

func TestIsH264Codec(t *testing.T) {
	cases := []struct {
		codecs string
		want   bool
	}{
		{"avc1.4d401e", true},
		{"avc1.640028", true},
		{"avc3.64001F", true},
		{"AVC1.640028", true}, // case-insensitive
		{"H264.640028", true},
		{"h264", true},
		{"vp09.00.40.08.00.01.01.01.00", false},
		{"vp8.", false},
		{"av01.0.05M.08", false},
		{"hev1.1.6.L93.B0", false},
		{"hvc1.1.6.L93.B0", false},
		{"mp4a.40.5", false},
		{"", false}, // no codecs attribute → not selectable as H.264
	}
	for _, c := range cases {
		if got := isH264Codec(c.codecs); got != c.want {
			t.Errorf("isH264Codec(%q) = %v, want %v", c.codecs, got, c.want)
		}
	}
}
