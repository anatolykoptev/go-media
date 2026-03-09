package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	media "github.com/anatolykoptev/go-media"
)

// oxBackend delegates video extraction to ox-browser's /media/download endpoint.
// ox-browser handles page fetch, YouTube parser, download, and DASH merge.
type oxBackend struct {
	baseURL string
	client  *http.Client
}

// mediaDownloadRequest is the ox-browser /media/download API request.
type mediaDownloadRequest struct {
	URL       string `json:"url"`
	MediaType string `json:"media_type"`
	MaxHeight int    `json:"max_height,omitempty"`
	MaxSizeMB int    `json:"max_size_mb,omitempty"`
}

// mediaDownloadResponse is the ox-browser /media/download API response.
type mediaDownloadResponse struct {
	MediaType   string        `json:"media_type"`
	Files       []mediaFile   `json:"files"`
	Platform    string        `json:"platform"`
	Title       string        `json:"title"`
	Author      string        `json:"author"`
	Description string        `json:"description"`
	DurationSec *float64      `json:"duration_secs"`
	Stats       *mediaStats   `json:"stats"`
	Quality     *mediaQuality `json:"quality"`
	Merged      bool          `json:"merged"`
	Error       string        `json:"error"`
}

type mediaFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type mediaStats struct {
	Views    *int64 `json:"views"`
	Likes    *int64 `json:"likes"`
	Comments *int64 `json:"comments"`
}

type mediaQuality struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// extract calls ox-browser /media/download which handles everything:
// page fetch, YouTube playerResponse parsing, download, DASH merge.
func (b *oxBackend) extract(ctx context.Context, videoURL string) (*media.Media, error) {
	payload, _ := json.Marshal(mediaDownloadRequest{
		URL:       videoURL,
		MediaType: "video",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.baseURL+"/media/download", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("oxbrowser: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	cl := b.client
	if cl == nil {
		cl = http.DefaultClient
	}
	resp, err := cl.Do(req) //nolint:gosec // URL from trusted config
	if err != nil {
		return nil, fmt.Errorf("oxbrowser: unavailable: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= http.StatusBadRequest {
		var errResp mediaDownloadResponse
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("oxbrowser: %s (HTTP %d)", errResp.Error, resp.StatusCode)
	}

	var dr mediaDownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, fmt.Errorf("oxbrowser: decode: %w", err)
	}
	if len(dr.Files) == 0 {
		return nil, fmt.Errorf("oxbrowser: no files returned")
	}

	return mapToMedia(videoURL, &dr), nil
}

// mapToMedia converts ox-browser response to go-media Media struct.
func mapToMedia(videoURL string, dr *mediaDownloadResponse) *media.Media {
	m := &media.Media{
		Platform:    dr.Platform,
		URL:         videoURL,
		Title:       dr.Title,
		Author:      dr.Author,
		Description: dr.Description,
		LocalPath:   dr.Files[0].Path,
	}

	if dr.DurationSec != nil {
		m.Duration = time.Duration(*dr.DurationSec * float64(time.Second))
	}
	if dr.Stats != nil {
		if dr.Stats.Views != nil {
			m.Stats.Views = *dr.Stats.Views
		}
		if dr.Stats.Likes != nil {
			m.Stats.Likes = *dr.Stats.Likes
		}
		if dr.Stats.Comments != nil {
			m.Stats.Comments = *dr.Stats.Comments
		}
	}
	if dr.Quality != nil {
		label := fmt.Sprintf("%dp", dr.Quality.Height)
		m.Qualities = []media.Quality{{
			Label:  label,
			Width:  dr.Quality.Width,
			Height: dr.Quality.Height,
		}}
	}

	return m
}
