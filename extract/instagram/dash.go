package instagram

import (
	"fmt"

	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
	"github.com/anatolykoptev/go-media/extract/dash"
)

// populateMedia sets m.VideoURL/m.AudioURL/m.Qualities from the post. When a
// DASH manifest is present and parseable it picks the best video representation
// fitting maxSize plus the best audio representation (the processor's existing
// mergeDASH path muxes them). When the manifest is absent or unparseable it
// falls back to the video_versions list — the embed/SSR/proxy rungs never carry
// a manifest and must keep working exactly as before. It never returns an
// error: an unusable manifest degrades to video_versions rather than failing.
func populateMedia(m *media.Media, post threads.Post, maxSize int64) {
	if post.VideoDashManifest != "" {
		man, err := dash.ParseManifest(post.VideoDashManifest)
		if err == nil && len(man.Videos) > 0 && len(man.Audios) > 0 {
			video, audio, selErr := dash.Select(man, maxSize)
			// Require BOTH a non-empty video URL and a non-empty audio URL.
			// A manifest whose representations carry no BaseURL (or whose
			// selected rep has no URL) yields empty URLs — committing to the
			// DASH branch would hard-fail the download. Fall through to
			// video_versions exactly as the no-manifest path does.
			if selErr == nil && video.URL != "" && audio.URL != "" {
				applyDASH(m, man, video, audio)
				return
			}
		}
		// Unparseable / unusable manifest → fall back to video_versions.
	}

	applyVideoVersions(m, post)
}

// applyDASH populates m from the chosen DASH representations and lists every
// video representation as a Quality (with estimated size for observability).
func applyDASH(m *media.Media, man *dash.Manifest, video, audio dash.Representation) {
	m.VideoURL = video.URL
	m.AudioURL = audio.URL
	for _, r := range man.Videos {
		m.Qualities = append(m.Qualities, media.Quality{
			Label:  qualityLabel(r),
			URL:    r.URL,
			Width:  r.Width,
			Height: r.Height,
			Size:   dash.EstimatedSize(r, man.Duration),
		})
	}
}

// applyVideoVersions reproduces the pre-DASH behaviour: first entry is the
// download URL, every entry becomes a Quality.
func applyVideoVersions(m *media.Media, post threads.Post) {
	if len(post.Videos) == 0 {
		return
	}
	m.VideoURL = post.Videos[0].URL
	for _, v := range post.Videos {
		m.Qualities = append(m.Qualities, media.Quality{
			Label:  qualityLabelFromHeight(v.Height),
			URL:    v.URL,
			Width:  v.Width,
			Height: v.Height,
		})
	}
}

// qualityLabel prefers a manifest label, then the representation id, then a
// height-derived "Np" string.
func qualityLabel(r dash.Representation) string {
	if r.Label != "" {
		return r.Label
	}
	if r.ID != "" && !isNumeric(r.ID) {
		return r.ID
	}
	return qualityLabelFromHeight(r.Height)
}

func qualityLabelFromHeight(h int) string {
	if h <= 0 {
		return ""
	}
	return fmt.Sprintf("%dp", h)
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}
