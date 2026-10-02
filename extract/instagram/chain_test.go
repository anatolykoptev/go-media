package instagram

import (
	"strings"
	"testing"
	"time"

	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
)

// threePostChain is a complete 3-post author chain in writing order. The
// linked post is the MIDDLE one (chainCode2) so a test can prove the posts
// ABOVE the link survive the hand-off.
const (
	chainUser  = "johndoe"
	chainCode1 = "AAA111"
	chainCode2 = "BBB222" // linked post
	chainCode3 = "CCC333"
	// parseURLThreadsCode is the post code used by the URL-parsing tests,
	// kept as a const so goconst does not double-count it against the
	// literal in extractor_test.go.
	parseURLThreadsCode = "DEF456"
)

func threePostChain() *threads.Chain {
	return &threads.Chain{
		Username: chainUser,
		AuthorID: "999",
		Complete: true,
		Posts: []threads.Post{
			{Code: chainCode1, Text: "First post of the chain.", Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(1000)},
			{Code: chainCode2, Text: "Second post, the linked one.", MediaType: 2, Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(2000), Videos: []threads.MediaVersion{{URL: "https://cdn.example.com/linked.mp4", Width: 720, Height: 1280}}},
			{Code: chainCode3, Text: "Third post, the tail.", Author: threads.ThreadsUser{ID: "999", Username: chainUser}, CreatedAt: time.UnixMilli(3000)},
		},
	}
}

// TestApplyChainRendersAllPostsInOrder: a 3-post chain must land in
// Media.Description as one text, posts numbered [i/N] in writing order,
// separated by the RenderChain separator. Reverting applyChain to the old
// single-post Description (post.Text) drops posts 1 and 3 → RED.
func TestApplyChainRendersAllPostsInOrder(t *testing.T) {
	chain := threePostChain()
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode2, 0)

	if !strings.Contains(m.Description, "[1/3]") || !strings.Contains(m.Description, "First post of the chain.") {
		t.Fatalf("Description missing post 1: %q", m.Description)
	}
	if !strings.Contains(m.Description, "[2/3]") || !strings.Contains(m.Description, "Second post, the linked one.") {
		t.Fatalf("Description missing post 2: %q", m.Description)
	}
	if !strings.Contains(m.Description, "[3/3]") || !strings.Contains(m.Description, "Third post, the tail.") {
		t.Fatalf("Description missing post 3: %q", m.Description)
	}
	i1 := strings.Index(m.Description, "First post")
	i2 := strings.Index(m.Description, "Second post")
	i3 := strings.Index(m.Description, "Third post")
	if i1 >= i2 || i2 >= i3 {
		t.Fatalf("posts out of order in Description: %q", m.Description)
	}
}

// TestApplyChainMiddleLinkKeepsPostsAbove: a link at the MIDDLE of a chain
// must still surface the posts ABOVE it. GetAuthorChain reconstructs the
// whole chain regardless of which post was linked, so post 1 (above the
// linked post 2) must be present. Reverting to GetThread (ancestor path
// only, no continuations) would still surface post 1 here, but dropping
// the chain path entirely (Description = linked post text only) loses it.
func TestApplyChainMiddleLinkKeepsPostsAbove(t *testing.T) {
	chain := threePostChain()
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode2, 0)

	if !strings.Contains(m.Description, "First post of the chain.") {
		t.Fatalf("post ABOVE the link (post 1) missing from Description: %q", m.Description)
	}
}

// TestApplyChainIncompleteFlagReachesConsumer: Chain.Complete == false means
// "there may be more posts we could not reach". That fact must reach the
// consumer-visible output, not be dropped at the extractor boundary.
// RenderChain appends "[chain may be incomplete: <reason>]"; applyChain must
// put that into Description verbatim. Reverting the chain render drops the
// note → a truncated thread is presented as whole (the failure this work
// exists to prevent).
func TestApplyChainIncompleteFlagReachesConsumer(t *testing.T) {
	chain := threePostChain()
	chain.Complete = false
	chain.Reason = "reply page truncated (20 reply threads, cap 20); further author continuation may exist beyond the page"
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode2, 0)

	if !strings.Contains(m.Description, "[chain may be incomplete:") {
		t.Fatalf("Description missing incompleteness note: %q", m.Description)
	}
	if !strings.Contains(m.Description, chain.Reason) {
		t.Fatalf("Description missing completeness reason: %q", m.Description)
	}
	if m.Metadata["chain_complete"] != "false" {
		t.Fatalf("Metadata[chain_complete] = %q, want \"false\"", m.Metadata["chain_complete"])
	}
	if m.Metadata["chain_reason"] != chain.Reason {
		t.Fatalf("Metadata[chain_reason] = %q, want reason", m.Metadata["chain_reason"])
	}
}

// TestApplyChainCompleteSetsMetadata: a complete chain sets the programmatic
// flag so a consumer can gate without parsing the text.
func TestApplyChainCompleteSetsMetadata(t *testing.T) {
	chain := threePostChain() // Complete: true
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode2, 0)

	if m.Metadata["chain_complete"] != "true" {
		t.Fatalf("Metadata[chain_complete] = %q, want \"true\"", m.Metadata["chain_complete"])
	}
	if m.Metadata["chain_reason"] != "" {
		t.Fatalf("Metadata[chain_reason] = %q, want empty for complete chain", m.Metadata["chain_reason"])
	}
	if strings.Contains(m.Description, "[chain may be incomplete") {
		t.Fatalf("complete chain rendered an incompleteness note: %q", m.Description)
	}
}

// TestApplyChainSinglePostNoScaffolding: a single-post thread must render
// as one post with NO [1/1] prefix and NO "---" separator — just the text
// (plus a media note when media is present). Reverting to a naive
// per-post render that always prefixes would add scaffolding noise.
func TestApplyChainSinglePostNoScaffolding(t *testing.T) {
	chain := &threads.Chain{
		Username: chainUser,
		AuthorID: "999",
		Complete: true,
		Posts: []threads.Post{
			{Code: chainCode1, Text: "Solo post.", Author: threads.ThreadsUser{ID: "999", Username: chainUser}},
		},
	}
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode1, 0)

	if strings.Contains(m.Description, "[1/1]") {
		t.Fatalf("single-post chain has [1/1] prefix: %q", m.Description)
	}
	if strings.Contains(m.Description, "---") {
		t.Fatalf("single-post chain has separator: %q", m.Description)
	}
	if !strings.Contains(m.Description, "Solo post.") {
		t.Fatalf("single-post text missing: %q", m.Description)
	}
}

// TestApplyChainLinkedPostMediaPopulated: the linked post's downloadable
// media (VideoURL/Slides/Qualities) is populated from the LINKED post only —
// other chain posts' media is noted in the text but NOT fetched. Reverting
// to populateMedia(linkedPost) being skipped leaves VideoURL empty.
func TestApplyChainLinkedPostMediaPopulated(t *testing.T) {
	chain := threePostChain() // linked post (code2) is a video
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, chainCode2, 0)

	if m.VideoURL != "https://cdn.example.com/linked.mp4" {
		t.Fatalf("VideoURL = %q, want the linked post's video URL", m.VideoURL)
	}
	if m.Author != "@johndoe" {
		t.Fatalf("Author = %q, want @johndoe", m.Author)
	}
	if m.Metadata["code"] != chainCode2 {
		t.Fatalf("Metadata[code] = %q, want linked code %q", m.Metadata["code"], chainCode2)
	}
}

// TestApplyChainLinkedCodeNotFoundFallsBackToFirst: defensive — if the
// linked code is not in chain.Posts (should not happen per the walk, but
// the extractor must never panic), applyChain falls back to Posts[0] for
// media/author rather than crashing.
func TestApplyChainLinkedCodeNotFoundFallsBackToFirst(t *testing.T) {
	chain := threePostChain()
	m := &media.Media{Metadata: make(map[string]string)}

	applyChain(m, chain, "NOT_IN_CHAIN", 0)

	if m.Metadata["code"] != chainCode1 {
		t.Fatalf("fallback code = %q, want first post code %q", m.Metadata["code"], chainCode1)
	}
}

// TestParseURLThreadsCom: Meta migrated Threads to threads.com; the pattern
// must match both threads.net and threads.com and parse them identically.
// Reverting the host pattern to threads.net-only leaves threads.com URLs
// unmatched → the extractor misses them entirely.
func TestParseURLThreadsCom(t *testing.T) {
	t.Run("threads.com parses like threads.net", func(t *testing.T) {
		ig, tu, tc, err := parseURL("https://www.threads.com/@" + chainUser + "/post/" + parseURLThreadsCode)
		if err != nil {
			t.Fatalf("parseURL threads.com: unexpected error: %v", err)
		}
		if ig != "" || tu != chainUser || tc != parseURLThreadsCode {
			t.Fatalf("parseURL threads.com = (%q,%q,%q), want (\"\",%s,%s)", ig, tu, tc, chainUser, parseURLThreadsCode)
		}
	})
	t.Run("threads.net still parses", func(t *testing.T) {
		_, tu, tc, err := parseURL("https://www.threads.net/@" + chainUser + "/post/" + parseURLThreadsCode)
		if err != nil || tu != chainUser || tc != parseURLThreadsCode {
			t.Fatalf("parseURL threads.net regression: tu=%q tc=%q err=%v", tu, tc, err)
		}
	})
}

// TestMatchThreadsCom: Match must accept threads.com URLs.
func TestMatchThreadsCom(t *testing.T) {
	e := &Extractor{}
	if !e.Match("https://www.threads.com/@user/post/ABC123") {
		t.Fatal("Match(threads.com) = false, want true")
	}
	if !e.Match("https://www.threads.net/@user/post/ABC123") {
		t.Fatal("Match(threads.net) = false, want true (regression)")
	}
}
