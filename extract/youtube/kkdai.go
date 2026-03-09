package youtube

import (
	"context"
	"fmt"
	"sort"
	"strings"

	media "github.com/anatolykoptev/go-media"

	"github.com/kkdai/youtube/v2"
)

// kkdaiBackend wraps kkdai/youtube for metadata and direct URL extraction.
type kkdaiBackend struct{}

// extract tries to get video info and streaming URL using kkdai/youtube.
// Returns *media.Media with VideoURL (and optionally AudioURL for DASH).
// maxHeight: 0 = best quality, otherwise limit (e.g. 1080).
func (b *kkdaiBackend) extract(ctx context.Context, videoID string, maxHeight int) (*media.Media, error) {
	client := youtube.Client{}

	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("kkdai: get video: %w", err)
	}

	m := &media.Media{
		Platform:    "youtube",
		Title:       video.Title,
		Description: video.Description,
		Duration:    video.Duration,
		Metadata:    map[string]string{"video_id": videoID},
	}

	m.Qualities = kkdaiQualities(video.Formats)

	// Try combined format first (video + audio in one stream).
	if err := setCombinedStream(ctx, &client, video, m, maxHeight); err == nil {
		return m, nil
	}

	// Fall back to DASH (separate video + audio streams).
	if err := setDASHStreams(ctx, &client, video, m, maxHeight); err != nil {
		return nil, fmt.Errorf("kkdai: no suitable format: %w", err)
	}

	return m, nil
}

// setCombinedStream finds a muxed format (video+audio) and sets m.VideoURL.
func setCombinedStream(
	ctx context.Context,
	client *youtube.Client,
	video *youtube.Video,
	m *media.Media,
	maxHeight int,
) error {
	combined := video.Formats.
		Type("video/mp4").
		Select(func(f youtube.Format) bool {
			return f.AudioChannels > 0
		})

	f, ok := pickBestVideo(combined, maxHeight)
	if !ok {
		return fmt.Errorf("no combined format")
	}

	u, err := client.GetStreamURLContext(ctx, video, &f)
	if err != nil {
		return fmt.Errorf("stream URL: %w", err)
	}

	m.VideoURL = u

	return nil
}

// setDASHStreams finds separate video-only and audio-only formats.
func setDASHStreams(
	ctx context.Context,
	client *youtube.Client,
	video *youtube.Video,
	m *media.Media,
	maxHeight int,
) error {
	// Best video-only stream.
	videoOnly := video.Formats.
		Type("video/mp4").
		Select(func(f youtube.Format) bool {
			return f.AudioChannels == 0
		})

	vf, ok := pickBestVideo(videoOnly, maxHeight)
	if !ok {
		return fmt.Errorf("no video-only format")
	}

	videoURL, err := client.GetStreamURLContext(ctx, video, &vf)
	if err != nil {
		return fmt.Errorf("video stream URL: %w", err)
	}

	// Best audio-only stream.
	audioOnly := video.Formats.Select(func(f youtube.Format) bool {
		return f.AudioChannels > 0 && strings.Contains(f.MimeType, "audio/")
	})

	af, ok := pickBestAudio(audioOnly)
	if !ok {
		return fmt.Errorf("no audio-only format")
	}

	audioURL, err := client.GetStreamURLContext(ctx, video, &af)
	if err != nil {
		return fmt.Errorf("audio stream URL: %w", err)
	}

	m.VideoURL = videoURL
	m.AudioURL = audioURL

	return nil
}

// pickBestVideo returns the highest-resolution format within maxHeight.
func pickBestVideo(formats youtube.FormatList, maxHeight int) (youtube.Format, bool) {
	if len(formats) == 0 {
		return youtube.Format{}, false
	}

	candidates := make(youtube.FormatList, 0, len(formats))

	for _, f := range formats {
		if maxHeight <= 0 || f.Height <= maxHeight {
			candidates = append(candidates, f)
		}
	}

	if len(candidates) == 0 {
		return youtube.Format{}, false
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Height != candidates[j].Height {
			return candidates[i].Height > candidates[j].Height
		}
		return candidates[i].Bitrate > candidates[j].Bitrate
	})

	return candidates[0], true
}

// pickBestAudio returns the highest-bitrate audio format.
func pickBestAudio(formats youtube.FormatList) (youtube.Format, bool) {
	if len(formats) == 0 {
		return youtube.Format{}, false
	}

	best := formats[0]
	for _, f := range formats[1:] {
		if f.Bitrate > best.Bitrate {
			best = f
		}
	}

	return best, true
}

// kkdaiQualities converts kkdai video formats to media.Quality entries.
func kkdaiQualities(formats youtube.FormatList) []media.Quality {
	var qualities []media.Quality

	for _, f := range formats {
		if f.Width == 0 && f.Height == 0 {
			continue
		}

		qualities = append(qualities, media.Quality{
			Label:  f.QualityLabel,
			Width:  f.Width,
			Height: f.Height,
			Size:   f.ContentLength,
		})
	}

	return qualities
}
