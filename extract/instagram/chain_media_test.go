package instagram

import (
	"testing"
	"time"

	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
)

// photoPost builds a single-photo Threads post (MediaType 1) whose
// synthesised CarouselItem carries the photo-slide candidates used across
// the slide tests, so the chain path and the single-post path exercise the
// SAME buildSlide rendition selection.
func photoPost(code string, text string) threads.Post {
	return threads.Post{
		Code:      code,
		Text:      text,
		MediaType: 1,
		Author:    threads.ThreadsUser{ID: "999", Username: chainUser},
		CreatedAt: time.UnixMilli(1000),
		CarouselItems: []threads.CarouselItem{
			{MediaType: 1, Images: photoSlideCands()},
		},
	}
}

// carouselPost builds a 3-slide carousel Threads post (MediaType 8):
// [video, video, photo], in slide order, each slide carrying the same
// candidate sets the slide tests use.
func carouselPost(code string, text string) threads.Post {
	return threads.Post{
		Code:      code,
		Text:      text,
		MediaType: 8,
		Author:    threads.ThreadsUser{ID: "999", Username: chainUser},
		CreatedAt: time.UnixMilli(2000),
		CarouselItems: []threads.CarouselItem{
			{MediaType: 2, Videos: videoSlideVersions(), VideoDashManifest: h264SlideManifest, IsDashEligible: true},
			{MediaType: 2, Videos: videoSlideVersions(), VideoDashManifest: h264SlideManifest, IsDashEligible: true},
			{MediaType: 1, Images: photoSlideCands()},
		},
	}
}

// videoPost builds a single-video Threads post (MediaType 2) with the H.264
// DASH manifest fixture, so buildSlide (chain path) and populateVideoURL
// (linked-post path) both run dash.Select on the SAME manifest and must
// pick the SAME rendition.
func videoPost(code string, text string) threads.Post {
	return threads.Post{
		Code:              code,
		Text:              text,
		MediaType:         2,
		Author:            threads.ThreadsUser{ID: "999", Username: chainUser},
		CreatedAt:         time.UnixMilli(3000),
		Videos:            []threads.MediaVersion{{URL: fallbackVideoURL, Width: 720, Height: 1280}},
		VideoDashManifest: dashManifestFixture,
		CarouselItems: []threads.CarouselItem{
			{MediaType: 2, Videos: []threads.MediaVersion{{URL: fallbackVideoURL, Width: 720, Height: 1280}}, VideoDashManifest: dashManifestFixture, IsDashEligible: true},
		},
	}
}

// TestApplyChainCarriesPerPostMediaInOrder: a 3-post chain where posts 1
// and 3 carry photos and post 2 is text must associate each post's media
// with that post, in chain writing order, and post 2 must have none.
// Reverting applyChain to carry only the linked post's media leaves
// m.Posts nil → RED.
func TestApplyChainCarriesPerPostMediaInOrder(t *testing.T) {
	chain := &threads.Chain{
		Username: chainUser,
		AuthorID: "999",
		Complete: true,
		Posts: []threads.Post{
			photoPost(chainCode1, "First, with a photo."),
			{Code: chainCode2, Text: "Second, text only.", Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(2000), CarouselItems: []threads.CarouselItem{}},
			photoPost(chainCode3, "Third, with a photo."),
		},
	}
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode2, 0)

	if len(m.Posts) != 3 {
		t.Fatalf("len(Posts) = %d, want 3 (one per chain post, in order)", len(m.Posts))
	}
	// Post 1 — photo carried.
	if m.Posts[0].Index != 0 || m.Posts[0].Code != chainCode1 {
		t.Fatalf("Posts[0] = {Index:%d Code:%q}, want {0,%s}", m.Posts[0].Index, m.Posts[0].Code, chainCode1)
	}
	if len(m.Posts[0].Slides) != 1 || m.Posts[0].Slides[0].Type != media.SlideTypeImage {
		t.Fatalf("Posts[0].Slides = %+v, want 1 image slide", m.Posts[0].Slides)
	}
	if m.Posts[0].Slides[0].URL != photoHiResURL {
		t.Fatalf("Posts[0].Slides[0].URL = %q, want highest-res photo %q", m.Posts[0].Slides[0].URL, photoHiResURL)
	}
	// Post 2 — text only, no media, but the entry exists to keep indexing
	// aligned with chain order.
	if m.Posts[1].Index != 1 || m.Posts[1].Code != chainCode2 {
		t.Fatalf("Posts[1] = {Index:%d Code:%q}, want {1,%s}", m.Posts[1].Index, m.Posts[1].Code, chainCode2)
	}
	if len(m.Posts[1].Slides) != 0 {
		t.Fatalf("Posts[1].Slides = %+v, want empty (text-only post)", m.Posts[1].Slides)
	}
	// Post 3 — photo carried.
	if m.Posts[2].Index != 2 || m.Posts[2].Code != chainCode3 {
		t.Fatalf("Posts[2] = {Index:%d Code:%q}, want {2,%s}", m.Posts[2].Index, m.Posts[2].Code, chainCode3)
	}
	if len(m.Posts[2].Slides) != 1 || m.Posts[2].Slides[0].URL != photoHiResURL {
		t.Fatalf("Posts[2].Slides = %+v, want 1 photo slide at highest-res", m.Posts[2].Slides)
	}
}

// TestApplyChainCarouselPostAllSlidesInOrder: a chain post carrying a
// multi-slide carousel must surface every slide, in slide order, with the
// correct per-slide type. Reverting postSlides to flatten or drop slides
// loses order/type → RED.
func TestApplyChainCarouselPostAllSlidesInOrder(t *testing.T) {
	chain := &threads.Chain{
		Username: chainUser,
		AuthorID: "999",
		Complete: true,
		Posts:    []threads.Post{carouselPost(chainCode1, "Carousel post.")},
	}
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode1, 0)

	if len(m.Posts) != 1 {
		t.Fatalf("len(Posts) = %d, want 1", len(m.Posts))
	}
	if len(m.Posts[0].Slides) != 3 {
		t.Fatalf("len(Posts[0].Slides) = %d, want 3 (one per carousel slide)", len(m.Posts[0].Slides))
	}
	wantTypes := []media.SlideType{media.SlideTypeVideo, media.SlideTypeVideo, media.SlideTypeImage}
	for i, wt := range wantTypes {
		if m.Posts[0].Slides[i].Type != wt {
			t.Errorf("slide %d Type = %d, want %d (slide order must be preserved)", i, m.Posts[0].Slides[i].Type, wt)
		}
	}
	// Video slides went through dash.Select on the H.264 manifest → 1080p.
	if m.Posts[0].Slides[0].URL != slideDash1080pURL {
		t.Errorf("slide 0 URL = %q, want DASH H.264 1080p", m.Posts[0].Slides[0].URL)
	}
	// Photo slide (index 2) is the highest-res candidate.
	if m.Posts[0].Slides[2].URL != photoHiResURL {
		t.Errorf("slide 2 URL = %q, want highest-res photo", m.Posts[0].Slides[2].URL)
	}
}

// TestApplyChainVideoPostSameRenditionAsRoot: a chain post carrying a
// video must be carried with the SAME rendition choice the root/linked
// post's path makes. The linked post is the video post, so populateMedia
// sets m.VideoURL via populateVideoURL→dash.Select; postSlides sets
// Posts[0].Slides[0].URL via buildSlide→dash.Select on the SAME manifest.
// Both must pick the 1080p H.264 rep. Reverting postSlides to a second,
// laxer selection rule diverges → RED.
func TestApplyChainVideoPostSameRenditionAsRoot(t *testing.T) {
	chain := &threads.Chain{
		Username: chainUser,
		AuthorID: "999",
		Complete: true,
		Posts: []threads.Post{
			videoPost(chainCode1, "Video post."),
			{Code: chainCode2, Text: "Text follow-up.", Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(4000), CarouselItems: []threads.CarouselItem{}},
		},
	}
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode1, 0)

	// Linked-post path (the existing single-video seam).
	if m.VideoURL != dash1080pVideoURL {
		t.Fatalf("VideoURL = %q, want DASH 1080p (linked-post path)", m.VideoURL)
	}
	// Chain-path slide for the same video post — same dash.Select choice.
	if len(m.Posts) != 2 || len(m.Posts[0].Slides) != 1 {
		t.Fatalf("Posts = %+v, want 2 entries; Posts[0] want 1 video slide", m.Posts)
	}
	if m.Posts[0].Slides[0].Type != media.SlideTypeVideo {
		t.Fatalf("Posts[0].Slides[0].Type = %d, want SlideTypeVideo", m.Posts[0].Slides[0].Type)
	}
	if m.Posts[0].Slides[0].URL != m.VideoURL {
		t.Fatalf("Posts[0].Slides[0].URL = %q, want %q (SAME rendition as the root-post path)", m.Posts[0].Slides[0].URL, m.VideoURL)
	}
	if m.Posts[0].Slides[0].AudioURL != m.AudioURL {
		t.Fatalf("Posts[0].Slides[0].AudioURL = %q, want %q (SAME DASH audio as root-post path)", m.Posts[0].Slides[0].AudioURL, m.AudioURL)
	}
}

// TestApplyChainTextOnlyChainNoScaffolding: a text-only chain must produce
// no media and no empty Posts scaffolding (m.Posts nil). Reverting
// chainPostMedia to always emit one entry per post leaves a [{0,nil}]
// slice → RED.
func TestApplyChainTextOnlyChainNoScaffolding(t *testing.T) {
	chain := &threads.Chain{
		Username: chainUser,
		AuthorID: "999",
		Complete: true,
		Posts: []threads.Post{
			{Code: chainCode1, Text: "Just text.", Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(1000), CarouselItems: []threads.CarouselItem{}},
			{Code: chainCode2, Text: "More text.", Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(2000), CarouselItems: []threads.CarouselItem{}},
		},
	}
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode1, 0)

	if m.Posts != nil {
		t.Fatalf("Posts = %+v, want nil for a text-only chain (no scaffolding)", m.Posts)
	}
	if len(m.Slides) != 0 || m.VideoURL != "" {
		t.Fatalf("Slides/VideoURL non-empty for a text-only chain: %+v", m)
	}
}

// TestApplyChainTextMediaConsistency: the rendered-text media note and the
// carried media must agree. go-threads RenderChain emits a per-post
// mediaNote keyed off MediaType / CarouselItems / Images / Videos; the
// chain path builds Posts[i].Slides from the SAME CarouselItems (the
// synthesised per-slide view of exactly those fields). So the invariant
// reduces to: len(Posts) == len(chain.Posts) (one entry per post, in
// order) and len(Posts[i].Slides) == len(chain.Posts[i].CarouselItems)
// (every slide the text claims is carried, no more). A post the text
// calls text-only (empty CarouselItems) carries no slides; a post the
// text calls a carousel of N carries N slides. Reverting postSlides to
// drop or invent slides breaks the equality → RED.
func TestApplyChainTextMediaConsistency(t *testing.T) {
	chain := &threads.Chain{
		Username: chainUser,
		AuthorID: "999",
		Complete: true,
		Posts: []threads.Post{
			photoPost(chainCode1, "photo"), // [media: photo]
			{Code: chainCode2, Text: "text", Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(2000), CarouselItems: []threads.CarouselItem{}}, // text-only
			carouselPost(chainCode3, "carousel"), // [media: carousel, 3 slides]
			videoPost(chainCode1+"v", "video"),   // [media: video]
		},
	}
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode1, 0)

	if len(m.Posts) != len(chain.Posts) {
		t.Fatalf("len(Posts) = %d, want %d (one entry per chain post, in writing order)", len(m.Posts), len(chain.Posts))
	}
	for i, p := range chain.Posts {
		if m.Posts[i].Index != i {
			t.Errorf("Posts[%d].Index = %d, want %d", i, m.Posts[i].Index, i)
		}
		if m.Posts[i].Code != p.Code {
			t.Errorf("Posts[%d].Code = %q, want %q", i, m.Posts[i].Code, p.Code)
		}
		if len(m.Posts[i].Slides) != len(p.CarouselItems) {
			t.Errorf("Posts[%d]: len(Slides)=%d, want len(CarouselItems)=%d (text/media consistency: every claimed slide carried, no more)", i, len(m.Posts[i].Slides), len(p.CarouselItems))
		}
	}
}

// TestInstagramPathLeavesPostsNil: the chain path is Threads-only. The
// Instagram branch of ExtractWithBudget never calls applyChain, so it must
// leave m.Posts nil. This simulates that branch (the exact sequence
// extractor.go runs for an Instagram URL: Description = post.Text, then
// populateMedia) and asserts no Posts scaffolding is emitted — proving an
// Instagram URL is byte-identical to today.
func TestInstagramPathLeavesPostsNil(t *testing.T) {
	post := threads.Post{
		Code:              "IGABC123",
		Text:              "Instagram caption.",
		MediaType:         2,
		Videos:            []threads.MediaVersion{{URL: fallbackVideoURL, Width: 720, Height: 1280}},
		VideoDashManifest: dashManifestFixture,
	}
	m := &media.Media{
		Platform: "instagram",
		URL:      "https://www.instagram.com/reel/IGABC123/",
		Metadata: make(map[string]string),
	}
	// Mirror the Instagram branch of ExtractWithBudget verbatim.
	m.Description = post.Text
	m.Metadata["code"] = post.Code
	populateMedia(m, post, 0)

	if m.Posts != nil {
		t.Fatalf("Instagram path set Posts = %+v, want nil (chain scaffolding is Threads-only)", m.Posts)
	}
}
