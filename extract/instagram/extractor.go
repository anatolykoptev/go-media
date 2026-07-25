// Package instagram extracts video metadata from Instagram and Threads URLs
// using the go-threads client library.
package instagram

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
)

var urlPattern = regexp.MustCompile(
	`https?://(?:www\.)?(?:` +
		`instagram\.com/(?:p|reel)/([A-Za-z0-9_-]+)` +
		`|threads\.(?:net|com)/@([^/]+)/post/([A-Za-z0-9_-]+)` +
		`)`,
)

// Extractor implements media.Extractor for Instagram and Threads.
type Extractor struct {
	client *threads.Client
}

// New creates an Instagram/Threads extractor.
func New(client *threads.Client) *Extractor {
	return &Extractor{client: client}
}

func (e *Extractor) Name() string { return "instagram" }

func (e *Extractor) Match(url string) bool {
	return urlPattern.MatchString(url)
}

func (e *Extractor) Extract(ctx context.Context, rawURL string) (*media.Media, error) {
	return e.ExtractWithBudget(ctx, rawURL, 0)
}

// ExtractWithBudget fetches post metadata and picks the best DASH video
// representation that fits the byte budget (0 = no limit). When the post
// carries no DASH manifest (embed/SSR/proxy rungs) or the manifest is
// unparseable, it falls back to the video_versions behaviour unchanged.
//
// For Threads URLs the whole author chain is fetched (go-threads
// GetAuthorChain) and merged into one text via applyChain; for Instagram
// URLs the behaviour is byte-identical to the pre-chain implementation
// (GetInstagramPost, Description = post.Text, no chain scaffolding).
func (e *Extractor) ExtractWithBudget(ctx context.Context, rawURL string, maxSize int64) (*media.Media, error) {
	igCode, threadsUser, threadsCode, err := parseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("instagram: %w", err)
	}

	m := &media.Media{
		Platform: "instagram",
		URL:      rawURL,
		Metadata: make(map[string]string),
	}

	if threadsUser != "" {
		// Threads: fetch the whole author chain and merge it. The chain
		// carries every same-author post in writing order plus an honest
		// completeness flag; applyChain puts the rendered text into
		// Description and the linked post's downloadable media into the
		// single-video / slide slots. GetThread's reply threads are no
		// longer discarded.
		chain, err := e.client.GetAuthorChain(ctx, threadsUser, threadsCode)
		if err != nil {
			return nil, fmt.Errorf("instagram: fetch chain: %w", err)
		}
		if chain == nil || len(chain.Posts) == 0 {
			return nil, fmt.Errorf("instagram: no post data found")
		}
		applyChain(m, chain, threadsCode, maxSize)
		return m, nil
	}

	// Instagram: unchanged. GetInstagramPost + Description = post.Text, no
	// chain scaffolding — byte-identical to the pre-chain implementation.
	thread, err := e.client.GetInstagramPost(ctx, igCode)
	if err != nil {
		return nil, fmt.Errorf("instagram: fetch post: %w", err)
	}
	if thread == nil || len(thread.Items) == 0 {
		return nil, fmt.Errorf("instagram: no post data found")
	}

	post := thread.Items[0]
	m.Description = post.Text

	if post.Author.Username != "" {
		m.Author = "@" + post.Author.Username
		if post.Author.FullName != "" {
			m.Author = post.Author.FullName + " (@" + post.Author.Username + ")"
		}
	}
	m.Stats = mapStats(post)
	m.Metadata["code"] = igCode
	populateMedia(m, post, maxSize)

	return m, nil
}

// parseURL extracts shortcode/username from an Instagram or Threads URL.
func parseURL(rawURL string) (igCode, threadsUser, threadsCode string, err error) {
	m := urlPattern.FindStringSubmatch(rawURL)
	if m == nil {
		return "", "", "", errors.New("URL does not match Instagram/Threads pattern")
	}

	if m[1] != "" {
		return m[1], "", "", nil
	}
	if m[2] != "" && m[3] != "" {
		return "", m[2], m[3], nil
	}

	return "", "", "", errors.New("could not extract post code from URL")
}

// mapStats converts a go-threads Post's engagement counts into media.MediaStats.
// CommentCount (IG feed comments) is preferred; for Threads posts that only
// expose direct_reply_count, ReplyCount is used as a fallback when CommentCount
// is zero.
func mapStats(post threads.Post) media.MediaStats {
	comments := post.CommentCount
	if comments == 0 {
		comments = post.ReplyCount
	}
	return media.MediaStats{
		Views:    int64(post.ViewCount),
		Likes:    int64(post.LikeCount),
		Comments: int64(comments),
		Shares:   int64(post.RepostCount),
	}
}
