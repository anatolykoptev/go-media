package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	media "github.com/anatolykoptev/go-media"
	"github.com/lrstanley/go-ytdlp"
)

// defaultFormat is the yt-dlp format string preferring 1080p MP4 with M4A audio.
const defaultFormat = "bestvideo[height<=1080][ext=mp4]+bestaudio[ext=m4a]/" +
	"best[height<=1080][ext=mp4]/best"

// defaultExtractorArgs configures YouTube player client fallback.
const defaultExtractorArgs = "youtube:player_client=android,web"

// ytdlpBackend wraps go-ytdlp for reliable YouTube downloads.
type ytdlpBackend struct {
	binaryPath string // path to yt-dlp binary, empty = use PATH
}

// ytdlpInfo holds the subset of yt-dlp .info.json fields we need.
type ytdlpInfo struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Duration    *float64 `json:"duration"`
	Thumbnail   string   `json:"thumbnail"`
	Uploader    string   `json:"uploader"`
	ViewCount   *float64 `json:"view_count"`
	LikeCount   *float64 `json:"like_count"`
}

// download uses yt-dlp to download the video to outputPath.
// Returns *media.Media with LocalPath set (file already downloaded).
func (b *ytdlpBackend) download(
	ctx context.Context,
	videoURL, outputPath string,
	cfg Config,
) (*media.Media, error) {
	cmd := b.buildCommand(outputPath, cfg)

	result, err := cmd.Run(ctx, videoURL)
	if err != nil {
		return nil, fmt.Errorf("ytdlp: download failed: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("ytdlp: exit code %d: %s", result.ExitCode, result.Stderr)
	}

	m := &media.Media{
		Platform:  "youtube",
		URL:       videoURL,
		LocalPath: outputPath,
	}

	infoPath := outputPath + ".info.json"
	b.populateFromInfoJSON(m, infoPath)

	return m, nil
}

// buildCommand creates a configured ytdlp.Command.
func (b *ytdlpBackend) buildCommand(outputPath string, cfg Config) *ytdlp.Command {
	cmd := ytdlp.New().
		Format(defaultFormat).
		Output(outputPath).
		NoPlaylist().
		WriteInfoJSON().
		ExtractorArgs(defaultExtractorArgs)

	if b.binaryPath != "" {
		cmd.SetExecutable(b.binaryPath)
	}

	if cfg.CookiesFile != "" {
		cmd.Cookies(cfg.CookiesFile)
	}

	if cfg.Proxy != "" {
		cmd.Proxy(cfg.Proxy)
	}

	return cmd
}

// populateFromInfoJSON reads the .info.json sidecar and fills Media fields.
// The info file is removed after reading; errors are silently ignored
// because the download itself already succeeded.
func (b *ytdlpBackend) populateFromInfoJSON(m *media.Media, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	defer os.Remove(path) //nolint:errcheck

	var info ytdlpInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return
	}

	m.Title = info.Title
	m.Description = info.Description

	if info.Duration != nil {
		m.Duration = time.Duration(*info.Duration * float64(time.Second))
	}

	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}

	if info.Thumbnail != "" {
		m.Metadata["thumbnail"] = info.Thumbnail
	}

	m.Author = info.Uploader
	if info.ViewCount != nil {
		m.Stats.Views = int64(*info.ViewCount)
	}
	if info.LikeCount != nil {
		m.Stats.Likes = int64(*info.LikeCount)
	}
}
