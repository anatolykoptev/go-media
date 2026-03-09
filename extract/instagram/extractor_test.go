package instagram

import (
	"testing"
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
