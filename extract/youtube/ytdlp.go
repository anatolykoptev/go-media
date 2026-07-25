package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	// run executes the built yt-dlp command. Defaults to (*Command).Run;
	// overridable in tests to avoid shelling out to a real yt-dlp binary.
	run func(ctx context.Context, cmd *ytdlp.Command, videoURL string) (*ytdlp.Result, error)
}

// ytdlpInfo holds the subset of yt-dlp .info.json fields we need.
type ytdlpInfo struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Duration     *float64 `json:"duration"`
	Thumbnail    string   `json:"thumbnail"`
	Uploader     string   `json:"uploader"`
	ViewCount    *float64 `json:"view_count"`
	LikeCount    *float64 `json:"like_count"`
	CommentCount *float64 `json:"comment_count"`
	RepostCount  *float64 `json:"repost_count"`
}

// download uses yt-dlp to download the video to outputPath.
// Returns *media.Media with LocalPath set (file already downloaded). maxSize
// (bytes, 0 = no limit) is passed to yt-dlp as --max-filesize so an oversized
// video is aborted by yt-dlp instead of downloaded in full only to fail later.
func (b *ytdlpBackend) download(
	ctx context.Context,
	videoURL, outputPath string,
	cfg Config,
	maxSize int64,
) (*media.Media, error) {
	cmd := b.buildCommand(outputPath, cfg, maxSize)

	runFn := b.run
	if runFn == nil {
		runFn = func(ctx context.Context, c *ytdlp.Command, u string) (*ytdlp.Result, error) {
			return c.Run(ctx, u)
		}
	}
	result, err := runFn(ctx, cmd, videoURL)
	if err != nil {
		return nil, fmt.Errorf("ytdlp: download failed: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("ytdlp: exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// --max-filesize is a FILTER, not an abort: when the media exceeds the
	// budget yt-dlp SKIPS the download and still exits 0, leaving the output
	// path unwritten. It is also a no-op for HLS/DASH formats (which covers
	// most YouTube downloads), so the exit code alone is not a reliable
	// success signal. Stat the output file and surface a clear error if it
	// is missing or empty, regardless of exit code.
	info, statErr := os.Stat(outputPath)
	if statErr != nil || info.Size() == 0 {
		return nil, fmt.Errorf("ytdlp: download produced no output file %q (video exceeded size budget or --max-filesize no-op for this format)", outputPath)
	}

	m := &media.Media{
		Platform:  platformName,
		URL:       videoURL,
		LocalPath: outputPath,
	}

	infoPath := outputPath + ".info.json"
	if err := b.populateFromInfoJSON(m, infoPath); err != nil {
		// Download succeeded but metadata extraction failed — return the
		// media with a warning wrapped in the error so the consumer can
		// decide whether to use partial metadata or retry.
		return m, fmt.Errorf("ytdlp: download succeeded but metadata extraction failed: %w", err)
	}

	return m, nil
}

// buildCommand creates a configured ytdlp.Command. maxSize (bytes, 0 = no
// limit) adds --max-filesize so yt-dlp aborts an oversized download early.
func (b *ytdlpBackend) buildCommand(outputPath string, cfg Config, maxSize int64) *ytdlp.Command {
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

	if maxSize > 0 {
		// Suffixless value is interpreted by yt-dlp as bytes.
		cmd.MaxFileSize(strconv.FormatInt(maxSize, 10))
	}

	return cmd
}

// populateFromInfoJSON reads the .info.json sidecar and fills Media fields.
// The info file is removed after reading. Returns an error if the file
// cannot be read or parsed — the download itself already succeeded, so the
// caller should treat this as a non-fatal metadata warning.
func (b *ytdlpBackend) populateFromInfoJSON(m *media.Media, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read info json: %w", err)
	}
	defer os.Remove(path) //nolint:errcheck

	var info ytdlpInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return fmt.Errorf("parse info json: %w", err)
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
	if info.CommentCount != nil {
		m.Stats.Comments = int64(*info.CommentCount)
	}
	if info.RepostCount != nil {
		m.Stats.Shares = int64(*info.RepostCount)
	}

	return nil
}
