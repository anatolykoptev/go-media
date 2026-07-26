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

// postSlides builds the per-slide media for one chain post by ranging its
// CarouselItems through buildSlide — the SAME rendition-selection helper
// the linked/root post's carousel uses (dash.Select for video slides,
// bestImageVersion for photo slides, bestVideoVersion fallback). go-threads
// synthesises CarouselItems as a one-item list for a single-media post and
// a non-nil empty slice for a text-only post (buildCarouselItems), so this
// ONE code path covers carousel, single-photo, single-video, and text-only
// without a second Images/Videos derivation. Ordering is slide order. A
// slide whose rendition is unusable is returned with an empty URL
// (buildSlide's contract) so the processor reports it as a failed slide
// rather than the extractor silently dropping the post's media.
func postSlides(post threads.Post, maxSize int64) []media.Slide {
	slides := make([]media.Slide, 0, len(post.CarouselItems))
	for _, ci := range post.CarouselItems {
		slides = append(slides, buildSlide(ci, maxSize))
	}
	return slides
}
