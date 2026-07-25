package dash

import (
	"fmt"
	"sort"
	"strings"
)

// isH264Codec reports whether a DASH Representation codecs attribute denotes an
// H.264/AVC video track. Telegram's mobile clients cannot decode VP9/VP8/AV1
// video tracks (blank picture, working audio), so the selector must only ever
// return H.264 representations. A representation is H.264 when its codecs
// value starts with "avc1", "avc3", or "h264" (case-insensitive). Everything
// else (vp09, vp8, av01, hev1, hvc1, mp4a.*, or empty) is NOT H.264.
func isH264Codec(codecs string) bool {
	c := strings.ToLower(codecs)
	return strings.HasPrefix(c, "avc1") ||
		strings.HasPrefix(c, "avc3") ||
		strings.HasPrefix(c, "h264")
}

// Select picks the best video representation that fits the byte budget and the
// best audio representation. budget == 0 means no limit (pick the highest
// resolution). If budget > 0 and no video representation fits, the SMALLEST
// video representation is chosen rather than failing — degrading beats
// returning nothing. Returns an error only if there are no video
// representations or no audio representations (the mux path needs both).
func Select(man *Manifest, budget int64) (video, audio Representation, err error) {
	if man == nil || len(man.Videos) == 0 {
		return Representation{}, Representation{}, fmt.Errorf("dash: no video representations")
	}
	if len(man.Audios) == 0 {
		return Representation{}, Representation{}, fmt.Errorf("dash: no audio representations")
	}

	video = pickVideo(man.Videos, man.Duration, budget)
	audio = pickAudio(man.Audios)
	return video, audio, nil
}

// pickVideo returns the highest-resolution H.264 video whose estimated size
// fits the budget, or the smallest H.264 video if none fit. budget == 0
// disables the limit. Only H.264/AVC representations (codecs starting with
// avc1/avc3/h264) are selectable: Telegram's mobile clients cannot decode
// VP9/VP8/AV1 video tracks, so returning a non-H.264 rep would ship a
// blank-picture video. If no URL-bearing H.264 representation exists the
// selector returns a zero Representation so the caller degrades (e.g. to the
// H.264 video_versions rendition) instead of returning a non-H.264 rep that
// hard-fails playback or an empty-URL rep that hard-fails the download.
func pickVideo(reps []Representation, duration float64, budget int64) Representation {
	usable := make([]Representation, 0, len(reps))
	for _, r := range reps {
		if r.URL == "" || !isH264Codec(r.Codecs) {
			continue
		}
		usable = append(usable, r)
	}
	if len(usable) == 0 {
		// No URL-bearing H.264 representation — signal "no usable video".
		return Representation{}
	}

	// Sort by height desc, then bandwidth desc — best first.
	sort.SliceStable(usable, func(i, j int) bool {
		if usable[i].Height != usable[j].Height {
			return usable[i].Height > usable[j].Height
		}
		return usable[i].Bandwidth > usable[j].Bandwidth
	})

	if budget <= 0 {
		return usable[0]
	}

	// If no representation carries a known size (no FBContentLength and no
	// duration → effectiveSize is 0 for every rep), the fits check below
	// would treat every size-0 rep as fitting and pick the HIGHEST
	// resolution, silently uncapping the budget. A budget is a hard cap;
	// with unknown sizes the only rep we can be confident fits is the
	// smallest, so pick it as the safe default.
	hasSize := false
	for _, r := range usable {
		if effectiveSize(r, duration) > 0 {
			hasSize = true
			break
		}
	}
	if !hasSize {
		return usable[len(usable)-1]
	}

	for _, r := range usable {
		if effectiveSize(r, duration) <= budget {
			return r
		}
	}
	// Nothing fits — degrade to the smallest (last after desc sort).
	return usable[len(usable)-1]
}

// effectiveSize returns the best known download size in bytes for a
// representation: the exact FBContentLength (ContentLength) when present, else
// the bandwidth*duration/8 estimate. Returns 0 when neither is available.
func effectiveSize(r Representation, duration float64) int64 {
	if r.ContentLength > 0 {
		return r.ContentLength
	}
	return EstimatedSize(r, duration)
}

// pickAudio returns the highest-bandwidth audio representation with a URL.
// If no representation carries a URL it returns a zero Representation so the
// caller can degrade rather than silently muxing against an empty URL (which
// would yield silent video-only output).
func pickAudio(reps []Representation) Representation {
	best := Representation{}
	for _, r := range reps {
		if r.URL == "" {
			continue
		}
		if best.URL == "" || r.Bandwidth > best.Bandwidth {
			best = r
		}
	}
	return best
}
