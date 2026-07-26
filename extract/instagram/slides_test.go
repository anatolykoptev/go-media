package instagram

import (
	"testing"

	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
)

// h264SlideManifest is a per-slide DASH MPD with H.264 video reps (240p→1080p)
// carrying BaseURLs plus one audio rep — the same shape as dashManifestFixture
// but used to prove buildSlide runs a video slide through dash.Select (the
// H.264-only selection path) rather than bypassing it.
const h264SlideManifest = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" mediaPresentationDuration="PT0H0M30.000S" type="static">
  <Period>
    <AdaptationSet mimeType="video/mp4" contentType="video">
      <Representation id="240p" width="426" height="240" bandwidth="80000" codecs="avc1.4d401e">
        <BaseURL>https://slide-cdn.example.com/240p.mp4</BaseURL>
      </Representation>
      <Representation id="720p" width="720" height="1280" bandwidth="1100000" codecs="avc1.4d401f">
        <BaseURL>https://slide-cdn.example.com/720p.mp4</BaseURL>
      </Representation>
      <Representation id="1080p" width="1080" height="1920" bandwidth="2301000" codecs="avc1.640028">
        <BaseURL>https://slide-cdn.example.com/1080p.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet mimeType="audio/mp4" contentType="audio">
      <Representation id="audio" bandwidth="64000" codecs="mp4a.40.2">
        <BaseURL>https://slide-cdn.example.com/audio.m4a</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

// photoHiResURL is the highest-resolution image candidate used across the
// photo-slide tests (extracted as a const so goconst is satisfied).
const photoHiResURL = "https://cdn.example.com/photo_1080.jpg"

// slideDash1080pURL is the H.264 1080p DASH rep URL the slide/chain tests
// assert buildSlide picks from h264SlideManifest (const so goconst is
// satisfied across the slide and chain-media tests).
const slideDash1080pURL = "https://slide-cdn.example.com/1080p.mp4"

// dash1080pVideoURL is the H.264 1080p DASH rep URL the single-video and
// chain tests assert populateVideoURL / buildSlide pick from
// dashManifestFixture (const so goconst is satisfied across the
// extractor, slide, and chain-media tests).
const dash1080pVideoURL = "https://video-edge-1080p.example.com/video.mp4"

// photoSlideCands mirrors carousel_photo_DFeH7jYt2tv's per-slide image
// candidates: 2 candidates, the first 1080x1350 (highest), the second smaller.
func photoSlideCands() []threads.MediaVersion {
	return []threads.MediaVersion{
		{URL: photoHiResURL, Width: 1080, Height: 1350},
		{URL: "https://cdn.example.com/photo_480.jpg", Width: 480, Height: 600},
	}
}

// videoSlideVersions mirrors carousel_video_DWO51c8kfFH's per-slide
// video_versions: 3 renditions, the first 720x900 (highest here).
func videoSlideVersions() []threads.MediaVersion {
	return []threads.MediaVersion{
		{URL: videoSlideHiResURL, Width: 720, Height: 900},
		{URL: "https://cdn.example.com/vv_480.mp4", Width: 480, Height: 600},
		{URL: "https://cdn.example.com/vv_360.mp4", Width: 360, Height: 450},
	}
}

// videoSlideHiResURL is the highest-resolution video_versions rendition used
// across the slide and chain-media tests (const so goconst is satisfied).
const videoSlideHiResURL = "https://cdn.example.com/vv_720.mp4"

// TestBuildSlidePhotoPicksHighestResolution: a photo slide must choose the
// highest-resolution image candidate (1080x1350), not the first.
func TestBuildSlidePhotoPicksHighestResolution(t *testing.T) {
	ci := threads.CarouselItem{MediaType: 1, Images: photoSlideCands()}
	s := buildSlide(ci, 0)
	if s.Type != media.SlideTypeImage {
		t.Fatalf("Type = %d, want SlideTypeImage", s.Type)
	}
	if s.URL != photoHiResURL {
		t.Fatalf("URL = %q, want highest-res candidate", s.URL)
	}
	if s.Width != 1080 || s.Height != 1350 {
		t.Fatalf("dims = %dx%d, want 1080x1350", s.Width, s.Height)
	}
	if s.AudioURL != "" {
		t.Fatalf("AudioURL = %q, want empty for photo slide", s.AudioURL)
	}
}

// TestBuildSlideVideoDASHSelectsH264: a video slide with an H.264 DASH manifest
// must go through dash.Select and pick the best H.264 rep (1080p) + audio —
// proving the codec-selection path is exercised, not bypassed.
func TestBuildSlideVideoDASHSelectsH264(t *testing.T) {
	ci := threads.CarouselItem{
		MediaType:         2,
		Videos:            videoSlideVersions(),
		VideoDashManifest: h264SlideManifest,
		NumberOfQualities: 3,
		IsDashEligible:    true,
	}
	s := buildSlide(ci, 0)
	if s.Type != media.SlideTypeVideo {
		t.Fatalf("Type = %d, want SlideTypeVideo", s.Type)
	}
	if s.URL != slideDash1080pURL {
		t.Fatalf("URL = %q, want DASH 1080p H.264 rep (dash.Select exercised)", s.URL)
	}
	if s.AudioURL != "https://slide-cdn.example.com/audio.m4a" {
		t.Fatalf("AudioURL = %q, want DASH audio rep", s.AudioURL)
	}
}

// TestBuildSlideVideoVP9ManifestFallsBackToVideoVersions: a video slide whose
// manifest is VP9-only must fall back to the H.264 video_versions rendition
// (the same discipline as a single video) — never a VP9 URL that renders blank
// on Telegram.
func TestBuildSlideVideoVP9ManifestFallsBackToVideoVersions(t *testing.T) {
	ci := threads.CarouselItem{
		MediaType:         2,
		Videos:            videoSlideVersions(),
		VideoDashManifest: vp9OnlyManifestFixture, // VP9-only, from extractor_test.go
		IsDashEligible:    true,
	}
	s := buildSlide(ci, 50_000_000)
	if s.Type != media.SlideTypeVideo {
		t.Fatalf("Type = %d, want SlideTypeVideo", s.Type)
	}
	// Must be the highest-resolution video_versions rendition (720x900), NOT
	// a VP9 manifest URL.
	if s.URL != videoSlideHiResURL {
		t.Fatalf("URL = %q, want video_versions H.264 fallback (manifest is VP9-only)", s.URL)
	}
	if s.AudioURL != "" {
		t.Fatalf("AudioURL = %q, want empty (no DASH mux on VP9-only fall-through)", s.AudioURL)
	}
}

// TestBuildSlideVideoNoManifestUsesVideoVersions: a video slide with no DASH
// manifest (embed/SSR/proxy rung) must use the video_versions rendition.
func TestBuildSlideVideoNoManifestUsesVideoVersions(t *testing.T) {
	ci := threads.CarouselItem{MediaType: 2, Videos: videoSlideVersions()}
	s := buildSlide(ci, 0)
	if s.URL != videoSlideHiResURL {
		t.Fatalf("URL = %q, want highest video_versions rendition", s.URL)
	}
	if s.AudioURL != "" {
		t.Fatalf("AudioURL = %q, want empty (no manifest, no mux)", s.AudioURL)
	}
}

// TestPopulateMediaPhotoCarousel: a 10-slide photo carousel
// (DFeH7jYt2tv shape) must produce 10 ordered image Slides, each the
// highest-resolution candidate for its slide.
func TestPopulateMediaPhotoCarousel(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	items := make([]threads.CarouselItem, 10)
	for i := range items {
		items[i] = threads.CarouselItem{MediaType: 1, Images: photoSlideCands()}
	}
	post := threads.Post{MediaType: 8, CarouselItems: items}

	populateMedia(m, post, 0)

	if len(m.Slides) != 10 {
		t.Fatalf("len(Slides) = %d, want 10", len(m.Slides))
	}
	for i, s := range m.Slides {
		if s.Type != media.SlideTypeImage {
			t.Errorf("slide %d Type = %d, want SlideTypeImage", i, s.Type)
		}
		if s.URL != photoHiResURL {
			t.Errorf("slide %d URL = %q, want highest-res candidate", i, s.URL)
		}
	}
	// A carousel must NOT set VideoURL — the carousel path owns downloads.
	if m.VideoURL != "" {
		t.Errorf("VideoURL = %q, want empty for carousel", m.VideoURL)
	}
}

// TestPopulateMediaVideoCarousel: a 2-slide video carousel
// (DWO51c8kfFH shape) must produce 2 video Slides, each H.264 (DASH-selected
// when the manifest has H.264 reps).
func TestPopulateMediaVideoCarousel(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	items := []threads.CarouselItem{
		{MediaType: 2, Videos: videoSlideVersions(), VideoDashManifest: h264SlideManifest, IsDashEligible: true},
		{MediaType: 2, Videos: videoSlideVersions(), VideoDashManifest: h264SlideManifest, IsDashEligible: true},
	}
	post := threads.Post{MediaType: 8, CarouselItems: items}

	populateMedia(m, post, 0)

	if len(m.Slides) != 2 {
		t.Fatalf("len(Slides) = %d, want 2", len(m.Slides))
	}
	for i, s := range m.Slides {
		if s.Type != media.SlideTypeVideo {
			t.Errorf("slide %d Type = %d, want SlideTypeVideo", i, s.Type)
		}
		if s.URL != slideDash1080pURL {
			t.Errorf("slide %d URL = %q, want DASH H.264 1080p", i, s.URL)
		}
		if s.AudioURL != "https://slide-cdn.example.com/audio.m4a" {
			t.Errorf("slide %d AudioURL = %q, want DASH audio", i, s.AudioURL)
		}
	}
}

// TestPopulateMediaMixedCarouselOrderAndType: a mixed carousel
// (DWO51c8kfFH shape: [video, video, photo]) must preserve slide order AND
// carry the correct per-slide type — the case the old flat Images/Videos list
// could not express.
func TestPopulateMediaMixedCarouselOrderAndType(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	items := []threads.CarouselItem{
		{MediaType: 2, Videos: videoSlideVersions(), VideoDashManifest: h264SlideManifest, IsDashEligible: true},
		{MediaType: 2, Videos: videoSlideVersions(), VideoDashManifest: h264SlideManifest, IsDashEligible: true},
		{MediaType: 1, Images: photoSlideCands()},
	}
	post := threads.Post{MediaType: 8, CarouselItems: items}

	populateMedia(m, post, 0)

	if len(m.Slides) != 3 {
		t.Fatalf("len(Slides) = %d, want 3", len(m.Slides))
	}
	wantTypes := []media.SlideType{media.SlideTypeVideo, media.SlideTypeVideo, media.SlideTypeImage}
	for i, wt := range wantTypes {
		if m.Slides[i].Type != wt {
			t.Errorf("slide %d Type = %d, want %d", i, m.Slides[i].Type, wt)
		}
	}
	// Slide 2 is a photo — no audio URL.
	if m.Slides[2].AudioURL != "" {
		t.Errorf("slide 2 AudioURL = %q, want empty (photo slide)", m.Slides[2].AudioURL)
	}
	if m.Slides[2].URL != photoHiResURL {
		t.Errorf("slide 2 URL = %q, want photo candidate", m.Slides[2].URL)
	}
}

// TestPopulateMediaSinglePhoto: a single photo post (MediaType 1) must
// populate m.Slides with one image Slide and NOT set VideoURL — so the
// processor's "no video URL found" guard does not fire for a photo.
func TestPopulateMediaSinglePhoto(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{
		MediaType: 1,
		CarouselItems: []threads.CarouselItem{
			{MediaType: 1, Images: photoSlideCands()},
		},
	}

	populateMedia(m, post, 0)

	if len(m.Slides) != 1 {
		t.Fatalf("len(Slides) = %d, want 1 (single photo)", len(m.Slides))
	}
	if m.Slides[0].Type != media.SlideTypeImage {
		t.Fatalf("Type = %d, want SlideTypeImage", m.Slides[0].Type)
	}
	if m.VideoURL != "" {
		t.Errorf("VideoURL = %q, want empty (single photo, no video)", m.VideoURL)
	}
}

// TestPopulateMediaSingleVideoUnchanged: a single video (MediaType 2) must
// keep the existing VideoURL/AudioURL path and NOT populate Slides — the
// single-video pipeline (transcription, mux, clips) depends on this.
func TestPopulateMediaSingleVideoUnchanged(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{
		MediaType:         2,
		Videos:            []threads.MediaVersion{{URL: fallbackVideoURL, Width: 720, Height: 1280}},
		VideoDashManifest: dashManifestFixture,
	}

	populateMedia(m, post, 0)

	if len(m.Slides) != 0 {
		t.Fatalf("len(Slides) = %d, want 0 (single video stays on VideoURL path)", len(m.Slides))
	}
	if m.VideoURL != dash1080pVideoURL {
		t.Fatalf("VideoURL = %q, want DASH 1080p (existing single-video path)", m.VideoURL)
	}
}

// TestPopulateMediaTextOnlyNoPanic: a text-only post (empty CarouselItems,
// no video) must not panic and must leave Slides empty.
func TestPopulateMediaTextOnlyNoPanic(t *testing.T) {
	m := &media.Media{Metadata: make(map[string]string)}
	post := threads.Post{MediaType: 8, CarouselItems: []threads.CarouselItem{}}

	populateMedia(m, post, 0)

	if len(m.Slides) != 0 {
		t.Fatalf("len(Slides) = %d, want 0 (text-only)", len(m.Slides))
	}
}
