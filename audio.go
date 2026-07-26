package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProbeDuration returns video/audio duration in seconds using ffprobe.
// Returns an error if ffprobe is not installed, fails, or the file has no duration.
func ProbeDuration(ctx context.Context, path string) (int, error) {
	probeCtx, cancel := context.WithTimeout(ctx, DefaultProbeTimeout)
	defer cancel()

	out, err := exec.CommandContext(probeCtx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	var dur float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &dur); err != nil {
		return 0, fmt.Errorf("parse duration: %w", err)
	}
	return int(dur) + 1, nil // round up
}

// ExtractAudioChunk extracts a WAV audio chunk from a video file using ffmpeg.
// Output is 16kHz mono PCM suitable for Whisper.
func ExtractAudioChunk(ctx context.Context, videoPath, outputPath string, offsetSec, durationSec int) error {
	ffCtx, cancel := context.WithTimeout(ctx, DefaultFFmpegTimeout)
	defer cancel()

	cmd := exec.CommandContext(ffCtx, "ffmpeg",
		"-i", videoPath,
		"-ss", fmt.Sprintf("%d", offsetSec),
		"-t", fmt.Sprintf("%d", durationSec),
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"-y",
		outputPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg extract chunk at %ds: %w", offsetSec, err)
	}
	return nil
}

// MergeDASH downloads a separate audio stream and muxes it with the video
// using ffmpeg, then replaces the original video-only file with the merged
// result. It is the single authoritative DASH-mux path used by both the
// pipeline's single-video and carousel-slide paths (via (*Processor).mergeDASH)
// and by external callers (e.g. vaelor-agent's Threads chain-post delivery).
//
// On a download or mux failure it returns (videoPath, err) — the original
// video-only file is still on disk, so the caller may degrade to video-only
// or treat the slide as failed. On a rename failure it returns (mergedPath,
// err): os.Remove(videoPath) has already deleted the original, so the muxed
// output exists only at mergedPath and the caller must recover it there (the
// videoPath slot is empty). The audio file is always cleaned up. On success
// the returned path equals videoPath (the merged file was renamed over the
// original).
func MergeDASH(ctx context.Context, client HTTPDoer, videoPath, audioURL string, maxSize int64) (string, error) {
	audioPath := videoPath + ".audio.m4a"
	if err := DownloadFile(ctx, client, audioURL, audioPath, maxSize); err != nil {
		return videoPath, fmt.Errorf("download audio: %w", err) //nolint:wrapcheck // already wrapped
	}
	defer cleanupFile(audioPath)

	mergedPath := videoPath + ".merged.mp4"
	if err := MergeAudioVideo(ctx, videoPath, audioPath, mergedPath); err != nil {
		return videoPath, err
	}

	// Replace original video-only file with merged.
	_ = os.Remove(videoPath) //nolint:errcheck // best-effort cleanup
	if err := os.Rename(mergedPath, videoPath); err != nil {
		return mergedPath, fmt.Errorf("rename merged file: %w", err)
	}
	return videoPath, nil
}

// MergeAudioVideo combines a video-only and audio-only file into a single MP4 using ffmpeg.
// Used for DASH streams where video and audio are separate.
func MergeAudioVideo(ctx context.Context, videoPath, audioPath, outputPath string) error {
	ffCtx, cancel := context.WithTimeout(ctx, DefaultFFmpegTimeout)
	defer cancel()

	cmd := exec.CommandContext(ffCtx, "ffmpeg",
		"-i", videoPath,
		"-i", audioPath,
		"-c", "copy",
		"-y",
		outputPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg merge audio+video: %w", err)
	}
	return nil
}

// ChunkAndTranscribe splits audio into chunks and transcribes each one.
// Returns (nil, nil) if transcriber is nil (opt-out).
// Returns (nil, error) if the transcriber is unavailable or ffprobe fails.
// Returns (partial, error) if some chunks failed — the partial result contains
// the successfully transcribed text and FailedChunks is set.
func ChunkAndTranscribe(ctx context.Context, videoPath, tempDir string, t Transcriber, opts Options) (*Transcription, error) {
	if t == nil {
		return nil, nil
	}
	if !t.Available() {
		return nil, fmt.Errorf("transcriber unavailable")
	}

	opts.defaults()
	duration, err := ProbeDuration(ctx, videoPath)
	if err != nil {
		return nil, fmt.Errorf("probe duration: %w", err)
	}
	if duration <= 0 {
		return nil, fmt.Errorf("probe duration: no duration in file")
	}

	base := filepath.Base(videoPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var chunks []Chunk
	var texts []string
	var failedChunks int

	for offset := 0; offset < duration; offset += opts.ChunkSec {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("%s_%d.wav", name, offset))

		if err := ExtractAudioChunk(ctx, videoPath, chunkPath, offset, opts.ChunkSec); err != nil {
			cleanupFile(chunkPath)
			failedChunks++
			continue
		}

		result, trErr := t.Transcribe(ctx, chunkPath)
		if trErr != nil {
			cleanupFile(chunkPath)
			failedChunks++
			continue
		}

		text := ""
		if result != nil {
			text = strings.TrimSpace(result.Text)
		}
		cleanupFile(chunkPath)

		if text == "" {
			continue
		}

		chunks = append(chunks, Chunk{
			Start: float64(offset),
			End:   float64(min(offset+opts.ChunkSec, duration)),
			Text:  text,
		})
		texts = append(texts, text)
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("transcription failed: all %d chunks failed", failedChunks)
	}

	tr := &Transcription{
		Text:         strings.Join(texts, " "),
		Duration:     float64(duration),
		Chunks:       chunks,
		FailedChunks: failedChunks,
	}

	if failedChunks > 0 {
		return tr, fmt.Errorf("transcription partial: %d/%d chunks failed", failedChunks, len(chunks)+failedChunks)
	}

	return tr, nil
}

func cleanupFile(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}
