// Package youtube extracts video metadata from YouTube URLs
// using a tiered backend approach: API → yt-dlp → ox-browser.
package youtube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	media "github.com/anatolykoptev/go-media"
)

// urlPattern matches all common YouTube URL formats:
//   - youtube.com/watch?v=XXX
//   - youtu.be/XXX
//   - youtube.com/shorts/XXX
//   - youtube.com/embed/XXX
//   - music.youtube.com/watch?v=XXX
//   - m.youtube.com/watch?v=XXX
var urlPattern = regexp.MustCompile(
	`https?://(?:` +
		`(?:www\.|m\.|music\.)?youtube\.com/(?:watch\?|shorts/|embed/)` +
		`|youtu\.be/` +
		`)`,
)

// videoIDPattern extracts an 11-character video ID.
var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// Config configures the YouTube extractor backends.
type Config struct {
	YtdlpPath    string // path to yt-dlp binary (empty = skip tier 2)
	OxBrowserURL string // ox-browser base URL (empty = skip tier 3)
	CookiesFile  string // path to cookies.txt for age-restricted content
	Proxy        string // HTTP proxy URL for downloads
	TempDir      string // temp directory for yt-dlp downloads
}

// maxVideoHeight is the default maximum video height for kkdai backend.
const maxVideoHeight = 1080

// Extractor implements media.Extractor for YouTube.
type Extractor struct {
	cfg  Config
	kkd  *kkdaiBackend
	ydlp *ytdlpBackend
	ox   *oxBackend
}

// New creates a YouTube extractor with the given configuration.
func New(cfg Config) *Extractor {
	e := &Extractor{
		cfg: cfg,
		kkd: &kkdaiBackend{},
	}
	if cfg.YtdlpPath != "" {
		e.ydlp = &ytdlpBackend{binaryPath: cfg.YtdlpPath}
	}
	if cfg.OxBrowserURL != "" {
		e.ox = &oxBackend{baseURL: cfg.OxBrowserURL, client: &http.Client{}}
	}
	return e
}

// Name returns the platform name.
func (e *Extractor) Name() string { return "youtube" }

// Match reports whether the URL is a YouTube video URL.
func (e *Extractor) Match(rawURL string) bool {
	return urlPattern.MatchString(rawURL)
}

// Extract fetches video metadata from a YouTube URL using a tiered fallback:
// 1. kkdai/youtube (fast, pure Go)
// 2. go-ytdlp (reliable, handles POT/anti-bot)
// 3. ox-browser (CF bypass, page parsing)
func (e *Extractor) Extract(ctx context.Context, rawURL string) (*media.Media, error) {
	videoID, err := parseVideoID(rawURL)
	if err != nil {
		return nil, fmt.Errorf("youtube: %w", err)
	}

	// Tier 1: kkdai/youtube — fast, pure Go.
	m, err := e.kkd.extract(ctx, videoID, maxVideoHeight)
	if err == nil {
		m.URL = rawURL
		return m, nil
	}

	// Tier 2: go-ytdlp — reliable subprocess download.
	if e.ydlp != nil {
		outputPath := e.ytdlpOutputPath(videoID)
		m, ytErr := e.ydlp.download(ctx, rawURL, outputPath, e.cfg)
		if ytErr == nil {
			return m, nil
		}
		err = fmt.Errorf("%w; ytdlp: %w", err, ytErr)
	}

	// Tier 3: ox-browser — page fetch + player response parsing.
	if e.ox != nil {
		m, oxErr := e.ox.extract(ctx, rawURL)
		if oxErr == nil {
			return m, nil
		}
		err = fmt.Errorf("%w; %w", err, oxErr)
	}

	return nil, fmt.Errorf("youtube: all backends failed: %w", err)
}

// ytdlpOutputPath returns the output path for yt-dlp downloads.
func (e *Extractor) ytdlpOutputPath(videoID string) string {
	dir := e.cfg.TempDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "go-media")
	}
	_ = os.MkdirAll(dir, 0o750) //nolint:mnd
	return filepath.Join(dir, "youtube_"+videoID+".mp4")
}

// parseVideoID extracts the video ID from any supported YouTube URL format.
func parseVideoID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	var id string

	switch {
	case u.Host == "youtu.be" || u.Host == "www.youtu.be":
		// youtu.be/VIDEO_ID
		id = strings.TrimPrefix(u.Path, "/")

	case strings.Contains(u.Host, "youtube.com"):
		path := u.Path
		switch {
		case strings.HasPrefix(path, "/watch"):
			id = u.Query().Get("v")
		case strings.HasPrefix(path, "/shorts/"):
			id = strings.TrimPrefix(path, "/shorts/")
		case strings.HasPrefix(path, "/embed/"):
			id = strings.TrimPrefix(path, "/embed/")
		}
	}

	// Strip trailing slashes or query fragments from path-based IDs.
	id = strings.TrimRight(id, "/")

	if id == "" {
		return "", errors.New("could not extract video ID from URL")
	}

	if !videoIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid video ID: %q", id)
	}

	return id, nil
}
