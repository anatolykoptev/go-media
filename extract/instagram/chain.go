package instagram

import (
	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
)

// applyChain merges a Threads author chain into the extracted Media for a
// Threads URL. It is the chain-aware replacement for the single-post path
// that ExtractWithBudget used to take for Threads URLs (GetThread +
// Description = post.Text).
//
// Decisions (see the task report for the full justification):
//
//   - FIELD: the merged chain text lands in m.Description. The consumer
//     renders m.Description into both the user-facing caption and the
//     LLM-facing block, so the merged text must live where a reader
//     actually sees it. m.Title is left empty (Threads posts have no
//     separate title) and no new field is added to Media — Description is
//     the existing, consumer-rendered slot.
//
//   - COMPLETENESS: Chain.Complete == false means "there may be more posts
//     we could not reach". RenderChain already appends
//     "[chain may be incomplete: <reason>]" to the text, so putting its
//     output in Description carries that fact to the consumer verbatim —
//     a truncated thread is never presented as whole. Additionally the
//     boolean and reason are written to m.Metadata ("chain_complete",
//     "chain_reason") so a programmatic consumer can gate without parsing
//     the text. Both: the text note is the surviving human/LLM signal, the
//     Metadata keys are the cheap programmatic seam.
//
//   - CHAIN MEDIA: the LINKED post's downloadable media is populated into
//     VideoURL/AudioURL/Qualities/Slides via populateMedia (the single-video
//     / single-carousel pipeline seam), AND every chain post's media is
//     carried into m.Posts in writing order via chainPostMedia (the chain-
//     wide seam). A consumer answers "which media belongs to post i" by
//     indexing m.Posts[i].Slides. RenderChain notes each post's media
//     presence in the text ("[media: video]" etc.); chainPostMedia builds
//     each post's slides from the SAME CarouselItems the text note is keyed
//     off, so a reader who sees "[media: photo]" actually gets the photo
//     and a text-only post claims nothing. The linked post's media appears
//     in BOTH seams (single-post pipeline + chain view); the redundancy is
//     intentional. m.Posts is nil for a text-only chain (no scaffolding).
//
//   - LINKED POST: the post whose code matches linkedCode (the code parsed
//     from the URL). It is always in chain.Posts (the walk includes the
//     linked post). If it is somehow absent, fall back to Posts[0] rather
//     than panic — never crash the extractor on a malformed chain.
func applyChain(m *media.Media, chain *threads.Chain, linkedCode string, maxSize int64) {
	linked := linkedChainPost(chain, linkedCode)

	// Merged chain text + completeness note -> the consumer-rendered slot.
	m.Description = threads.RenderChain(chain)

	// Programmatic completeness seam (the text note is the human/LLM seam).
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	if chain.Complete {
		m.Metadata["chain_complete"] = "true"
	} else {
		m.Metadata["chain_complete"] = "false"
		m.Metadata["chain_reason"] = chain.Reason
	}

	// Author / stats / code / downloadable media come from the linked post,
	// exactly as the pre-chain path did for the single post.
	if linked.Author.Username != "" {
		m.Author = "@" + linked.Author.Username
		if linked.Author.FullName != "" {
			m.Author = linked.Author.FullName + " (@" + linked.Author.Username + ")"
		}
	}
	m.Stats = mapStats(linked)
	m.Metadata["code"] = linked.Code

	populateMedia(m, linked, maxSize)

	// Carry every chain post's media, associated with its post, in chain
	// writing order — so a consumer can answer "which media belongs to post
	// i" by indexing m.Posts[i].Slides. The linked post's media ALSO remains
	// in m.VideoURL/m.Slides (populated above) for the single-video
	// pipeline; m.Posts is the chain-wide seam. nil for a text-only chain.
	m.Posts = chainPostMedia(chain, maxSize)
}

// linkedChainPost returns the chain post whose Code matches linkedCode, or
// chain.Posts[0] if none matches (defensive — the walk always includes the
// linked post; the fallback keeps the extractor from panicking on a
// malformed chain). Returns a zero Post when the chain has no posts.
func linkedChainPost(chain *threads.Chain, linkedCode string) threads.Post {
	if chain == nil || len(chain.Posts) == 0 {
		return threads.Post{}
	}
	for _, p := range chain.Posts {
		if p.Code == linkedCode {
			return p
		}
	}
	return chain.Posts[0]
}

// chainPostMedia builds the per-post media for every post in a Threads
// chain, in writing order (chain.Posts). Each PostMedia carries its 0-based
// chain index (matching RenderChain's [i/N] prefix), the post's code, and
// the post's slides built by postSlides. The returned slice has one entry
// per chain post so a consumer can index by chain position; a text-only
// post within a mixed chain keeps its entry (with nil Slides) so indexing
// stays aligned with the rendered text. A fully text-only chain returns
// nil — no media, no empty scaffolding.
func chainPostMedia(chain *threads.Chain, maxSize int64) []media.PostMedia {
	if chain == nil || len(chain.Posts) == 0 {
		return nil
	}
	posts := make([]media.PostMedia, len(chain.Posts))
	anyMedia := false
	for i, p := range chain.Posts {
		slides := postSlides(p, maxSize)
		if len(slides) > 0 {
			anyMedia = true
		}
		posts[i] = media.PostMedia{Index: i, Code: p.Code, Slides: slides}
	}
	if !anyMedia {
		return nil
	}
	return posts
}

// postSlides builds the per-slide media for one chain post. It ranges the
// post's CarouselItems through buildSlide — the SAME rendition-selection
// helper the linked/root post's carousel uses (dash.Select for video
// slides, bestImageVersion for photo slides, bestVideoVersion fallback).
// go-threads synthesises CarouselItems as a one-item list for a
// single-media post and a non-nil empty slice for a text-only post
// (buildCarouselItems, parsers.go:598), so that ONE code path covers
// carousel, single-photo, single-video, and text-only.
//
// FALLBACK: the go-threads instagram.go embed/SSR/proxy rungs
// (instagram.go:432-441, :500-507) set post.MediaType + post.Images /
// post.Videos DIRECTLY and never synthesise CarouselItems (only the
// graphql convertPost path does, via buildCarouselItems). On those posts
// CarouselItems is empty while the flattened lists carry the media — yet
// mediaNote (go-threads chain.go:423-440) still advertises
// "[media: photo]" / "[media: video]" off MediaType, producing exactly the
// text/media mismatch this branch exists to remove. When CarouselItems is
// empty but Images/Videos are not, fall back to the flattened lists by
// routing the post's own media through the SAME buildSlide helper
// (synthesising one CarouselItem from the post's fields) — no second
// rendition-selection rule. For a photo this calls bestImageVersion,
// mirroring populateMedia's flat-image fallback (dash.go:37); for a video
// it runs dash.Select then bestVideoVersion, the carousel video-slide
// rule. A post with CarouselItems populated takes ONLY the carousel branch
// — parsers.go:563-572 already flattens carousel slides into Images/Videos,
// so a naive union would duplicate every slide. A slide whose rendition is
// unusable is returned with an empty URL (buildSlide's contract) so the
// processor reports it as a failed slide rather than the extractor
// silently dropping the post's media.
func postSlides(post threads.Post, maxSize int64) []media.Slide {
	if len(post.CarouselItems) > 0 {
		slides := make([]media.Slide, 0, len(post.CarouselItems))
		for _, ci := range post.CarouselItems {
			slides = append(slides, buildSlide(ci, maxSize))
		}
		return slides
	}
	// CarouselItems empty — text-only post, or the instagram.go flat path
	// (MediaType + Images/Videos set, CarouselItems never synthesised).
	// No media at all → no slides, so the invariant
	// mediaNote != "" ⟺ ≥1 slide holds in both directions.
	if len(post.Images) == 0 && len(post.Videos) == 0 {
		return nil
	}
	// Flat media present — one CarouselItem synthesised from the post's own
	// fields = one slide for one flat post. No double count: this branch
	// only runs when CarouselItems is empty.
	ci := threads.CarouselItem{
		MediaType:         post.MediaType,
		Images:            post.Images,
		Videos:            post.Videos,
		VideoDashManifest: post.VideoDashManifest,
		NumberOfQualities: post.NumberOfQualities,
		IsDashEligible:    post.IsDashEligible,
	}
	// mediaNote's default branch (MediaType not 1/2/8) still returns
	// "[media: attached]" when Images/Videos are present; route by which
	// list is present so the slide type matches the text.
	if ci.MediaType != mediaTypeImage && ci.MediaType != mediaTypeVideo {
		if len(post.Videos) > 0 {
			ci.MediaType = mediaTypeVideo
		} else {
			ci.MediaType = mediaTypeImage
		}
	}
	return []media.Slide{buildSlide(ci, maxSize)}
}
