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
//   - CHAIN MEDIA: only the LINKED post's downloadable media is populated
//     (VideoURL/AudioURL/Qualities/Slides via populateMedia). Other chain
//     posts' media is NOT fetched as downloadable URLs — RenderChain notes
//     each post's media presence in the text ("[media: video]" etc.) so
//     the rendered text never implies a media-bearing post was text-only,
//     and never implies a non-linked post's media was fetched. Under-
//     promising is correct; a silent mismatch is not.
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
