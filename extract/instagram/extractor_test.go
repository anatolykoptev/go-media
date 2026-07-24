package instagram

import (
	"testing"

	threads "github.com/anatolykoptev/go-threads"

	"github.com/anatolykoptev/go-media"
)

func TestMatch(t *testing.T) {
	e := &Extractor{}

	tests := []struct {
		url  string
		want bool
	}{
		{"https://www.instagram.com/reel/ABC123/", true},
		{"https://instagram.com/p/XYZ789/", true},
		{"https://www.threads.net/@user/post/ABC123", true},
		{"https://youtube.com/watch?v=123", false},
		{"https://example.com/video", false},
		{"not a url", false},
	}

	for _, tt := range tests {
		if got := e.Match(tt.url); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestParseURL(t *testing.T) {
	tests := []struct {
		url         string
		igCode      string
		threadsUser string
		threadsCode string
		wantErr     bool
	}{
		{
			url:    "https://www.instagram.com/reel/ABC123/",
			igCode: "ABC123",
		},
		{
			url:    "https://instagram.com/p/XYZ-789_abc/",
			igCode: "XYZ-789_abc",
		},
		{
			url:         "https://www.threads.net/@johndoe/post/DEF456",
			threadsUser: "johndoe",
			threadsCode: "DEF456",
		},
		{
			url:     "https://youtube.com/watch?v=123",
			wantErr: true,
		},
		{
			url:     "not a url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		ig, tu, tc, err := parseURL(tt.url)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseURL(%q): expected error, got none", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseURL(%q): unexpected error: %v", tt.url, err)
			continue
		}
		if ig != tt.igCode {
			t.Errorf("parseURL(%q): igCode = %q, want %q", tt.url, ig, tt.igCode)
		}
		if tu != tt.threadsUser {
			t.Errorf("parseURL(%q): threadsUser = %q, want %q", tt.url, tu, tt.threadsUser)
		}
		if tc != tt.threadsCode {
			t.Errorf("parseURL(%q): threadsCode = %q, want %q", tt.url, tc, tt.threadsCode)
		}
	}
}

func TestMapStats(t *testing.T) {
	t.Run("all metrics", func(t *testing.T) {
		post := threads.Post{
			ViewCount:    56964765,
			LikeCount:    2244564,
			CommentCount: 24885,
			RepostCount:  27745,
			ReplyCount:   999, // ignored when CommentCount set
		}
		got := mapStats(post)
		want := media.MediaStats{
			Views:    56964765,
			Likes:    2244564,
			Comments: 24885,
			Shares:   27745,
		}
		if got != want {
			t.Fatalf("mapStats = %+v, want %+v", got, want)
		}
	})

	t.Run("comment fallback to reply count", func(t *testing.T) {
		// Threads posts expose direct_reply_count but no IG comment_count.
		post := threads.Post{
			LikeCount:    100,
			ReplyCount:   5,
			CommentCount: 0,
		}
		got := mapStats(post)
		if got.Comments != 5 {
			t.Fatalf("Comments = %d, want 5 (fallback from ReplyCount)", got.Comments)
		}
		if got.Likes != 100 {
			t.Errorf("Likes = %d, want 100", got.Likes)
		}
	})
}
