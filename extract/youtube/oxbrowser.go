package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	media "github.com/anatolykoptev/go-media"
)

var playerResponseRe = regexp.MustCompile(
	`var\s+ytInitialPlayerResponse\s*=\s*(\{.+?\})\s*;`,
)

// oxBackend extracts YouTube video URLs by fetching the page via ox-browser.
type oxBackend struct {
	baseURL string
	client  *http.Client
}

type fetchResponse struct {
	URL    string `json:"url"`
	Status int    `json:"status"`
	Body   string `json:"body"`
}

type playerResponse struct {
	VideoDetails struct {
		Title       string `json:"title"`
		Author      string `json:"author"`
		Description string `json:"shortDescription"`
		LengthSec   string `json:"lengthSeconds"`
		ViewCount   string `json:"viewCount"`
	} `json:"videoDetails"`
	StreamingData struct {
		Formats         []playerFormat `json:"formats"`
		AdaptiveFormats []playerFormat `json:"adaptiveFormats"`
	} `json:"streamingData"`
}

type playerFormat struct {
	ITag     int    `json:"itag"`
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Bitrate  int    `json:"bitrate"`
}

// extract fetches the YouTube page via ox-browser and parses the embedded player response.
func (b *oxBackend) extract(ctx context.Context, videoURL string) (*media.Media, error) {
	html, err := b.fetchPage(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	pr, err := parsePlayerResponse(html)
	if err != nil {
		return nil, err
	}
	return buildMedia(videoURL, pr)
}

func (b *oxBackend) fetchPage(ctx context.Context, videoURL string) (string, error) {
	payload, _ := json.Marshal(map[string]any{
		"url":          videoURL,
		"save_to_file": false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.baseURL+"/fetch-smart", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("oxbrowser: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	cl := b.client
	if cl == nil {
		cl = http.DefaultClient
	}
	resp, err := cl.Do(req) //nolint:gosec // URL from trusted config
	if err != nil {
		return "", fmt.Errorf("oxbrowser: unavailable: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	var fr fetchResponse
	if err := json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return "", fmt.Errorf("oxbrowser: decode response: %w", err)
	}
	if fr.Status != 0 && fr.Status >= 400 {
		return "", fmt.Errorf("oxbrowser: upstream status %d", fr.Status)
	}
	return fr.Body, nil
}

func parsePlayerResponse(html string) (*playerResponse, error) {
	m := playerResponseRe.FindStringSubmatch(html)
	if m == nil {
		return nil, fmt.Errorf("oxbrowser: player response not found")
	}
	var pr playerResponse
	if err := json.Unmarshal([]byte(m[1]), &pr); err != nil {
		return nil, fmt.Errorf("oxbrowser: parse player response: %w", err)
	}
	return &pr, nil
}

func buildMedia(videoURL string, pr *playerResponse) (*media.Media, error) {
	m := &media.Media{
		Platform:    "youtube",
		URL:         videoURL,
		Title:       pr.VideoDetails.Title,
		Author:      pr.VideoDetails.Author,
		Description: pr.VideoDetails.Description,
	}
	if sec, err := strconv.Atoi(pr.VideoDetails.LengthSec); err == nil {
		m.Duration = time.Duration(sec) * time.Second
	}
	if views, err := strconv.ParseInt(pr.VideoDetails.ViewCount, 10, 64); err == nil {
		m.Stats.Views = views
	}

	combined := filterDirect(pr.StreamingData.Formats)
	adaptive := filterDirect(pr.StreamingData.AdaptiveFormats)
	if len(combined) == 0 && len(adaptive) == 0 {
		return nil, fmt.Errorf("oxbrowser: no direct URLs available")
	}

	if len(combined) > 0 {
		m.VideoURL = pickBest(combined).URL
	} else {
		pickAdaptive(m, adaptive)
	}
	m.Qualities = oxQualities(combined, adaptive)
	return m, nil
}

func filterDirect(fmts []playerFormat) []playerFormat {
	out := make([]playerFormat, 0, len(fmts))
	for _, f := range fmts {
		if f.URL != "" {
			out = append(out, f)
		}
	}
	return out
}

func pickBest(fmts []playerFormat) playerFormat {
	sort.Slice(fmts, func(i, j int) bool {
		if fmts[i].Height != fmts[j].Height {
			return fmts[i].Height > fmts[j].Height
		}
		return fmts[i].Bitrate > fmts[j].Bitrate
	})
	return fmts[0]
}

func pickAdaptive(m *media.Media, fmts []playerFormat) {
	var videos, audios []playerFormat
	for _, f := range fmts {
		switch {
		case strings.HasPrefix(f.MimeType, "video/"):
			videos = append(videos, f)
		case strings.HasPrefix(f.MimeType, "audio/"):
			audios = append(audios, f)
		}
	}
	if len(videos) > 0 {
		m.VideoURL = pickBest(videos).URL
	}
	if len(audios) > 0 {
		best := audios[0]
		for _, a := range audios[1:] {
			if a.Bitrate > best.Bitrate {
				best = a
			}
		}
		m.AudioURL = best.URL
	}
}

func oxQualities(combined, adaptive []playerFormat) []media.Quality {
	all := append(combined, adaptive...) //nolint:gocritic
	qs := make([]media.Quality, 0, len(all))
	for _, f := range all {
		if f.Height == 0 && !strings.HasPrefix(f.MimeType, "audio/") {
			continue
		}
		label := "audio"
		if f.Height > 0 {
			label = strconv.Itoa(f.Height) + "p"
		}
		qs = append(qs, media.Quality{Label: label, URL: f.URL, Width: f.Width, Height: f.Height})
	}
	return qs
}
