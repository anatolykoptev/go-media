package dash

import (
	"fmt"
	"sort"
)

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

// pickVideo returns the highest-resolution video whose estimated size fits the
// budget, or the smallest video if none fit. budget == 0 disables the limit.
func pickVideo(reps []Representation, duration float64, budget int64) Representation {
	usable := make([]Representation, 0, len(reps))
	for _, r := range reps {
		if r.URL == "" {
			continue
		}
		usable = append(usable, r)
	}
	if len(usable) == 0 {
		// Fall back to the raw list rather than returning a zero value.
		usable = reps
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

	for _, r := range usable {
		if EstimatedSize(r, float64(duration)) <= budget {
			return r
		}
	}
	// Nothing fits — degrade to the smallest (last after desc sort).
	return usable[len(usable)-1]
}

// pickAudio returns the highest-bandwidth audio representation with a URL.
func pickAudio(reps []Representation) Representation {
	best := reps[0]
	for _, r := range reps[1:] {
		if r.URL == "" {
			continue
		}
		if best.URL == "" || r.Bandwidth > best.Bandwidth {
			best = r
		}
	}
	return best
}
