package instagram

import (
	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
	"github.com/anatolykoptev/go-media/extract/dash"
)

// go-threads MediaType vocabulary (Post.MediaType / CarouselItem.MediaType):
// 1 = image, 2 = video, 8 = carousel container (Post only). Named here so the
// extractor does not sprinkle bare magic numbers through the slide routing.
const (
	mediaTypeImage    = 1
	mediaTypeVideo    = 2
	mediaTypeCarousel = 8
)

// buildSlide picks the best rendition for one carousel slide and returns it
// as a Slide. For a video slide it applies the SAME H.264-only DASH selection
// as a single video (dash.Select → pickVideo skips non-H.264 reps), falling
// back to the slide's video_versions rendition when the manifest is absent,
// unparseable, VP9-only, or has no usable BaseURLs — never a second, laxer
// path. For a photo slide it picks the highest-resolution image candidate.
// maxSize bounds the per-slide rendition (0 = no limit), mirroring the
// single-video selector. A slide with no usable URL is returned with an empty
// URL so the processor reports it as a failed slide rather than panicking.
func buildSlide(ci threads.CarouselItem, maxSize int64) media.Slide {
	if ci.MediaType == mediaTypeVideo { // video slide
		if ci.VideoDashManifest != "" {
			man, err := dash.ParseManifest(ci.VideoDashManifest)
			if err == nil && len(man.Videos) > 0 && len(man.Audios) > 0 {
				video, audio, selErr := dash.Select(man, maxSize)
				// Require both a non-empty video URL and a non-empty audio URL,
				// exactly as populateVideoURL does for a single video — an
				// empty-URL DASH rep would hard-fail the download.
				if selErr == nil && video.URL != "" && audio.URL != "" {
					return media.Slide{
						Type:     media.SlideTypeVideo,
						URL:      video.URL,
						AudioURL: audio.URL,
						Width:    video.Width,
						Height:   video.Height,
					}
				}
			}
		}
		// Fall back to video_versions (always H.264 on Instagram). Pick the
		// highest-resolution rendition — the same "best by resolution" rule
		// the DASH selector uses (pickVideo sorts by height desc).
		if best := bestVideoVersion(ci.Videos); best.URL != "" {
			return media.Slide{
				Type:   media.SlideTypeVideo,
				URL:    best.URL,
				Width:  best.Width,
				Height: best.Height,
			}
		}
		return media.Slide{Type: media.SlideTypeVideo}
	}

	// Photo slide — pick the highest-resolution image candidate.
	best := bestImageVersion(ci.Images)
	return media.Slide{
		Type:   media.SlideTypeImage,
		URL:    best.URL,
		Width:  best.Width,
		Height: best.Height,
	}
}

// bestImageVersion returns the highest-resolution image candidate (max height,
// then max width as a tiebreak). Returns a zero MediaVersion when there are no
// URL-bearing candidates.
func bestImageVersion(cands []threads.MediaVersion) threads.MediaVersion {
	var best threads.MediaVersion
	for _, c := range cands {
		if c.URL == "" {
			continue
		}
		if best.URL == "" || c.Height > best.Height ||
			(c.Height == best.Height && c.Width > best.Width) {
			best = c
		}
	}
	return best
}

// bestVideoVersion returns the highest-resolution video_versions rendition
// (max height, then max width). video_versions are H.264 on Instagram, so no
// codec filter is needed here (the VP9 hazard lives only in DASH manifests,
// which buildSlide handles via dash.Select). Returns a zero MediaVersion when
// there are no URL-bearing renditions.
func bestVideoVersion(versions []threads.MediaVersion) threads.MediaVersion {
	var best threads.MediaVersion
	for _, v := range versions {
		if v.URL == "" {
			continue
		}
		if best.URL == "" || v.Height > best.Height ||
			(v.Height == best.Height && v.Width > best.Width) {
			best = v
		}
	}
	return best
}
