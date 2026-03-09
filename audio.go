package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ffmpegTimeout = 60 * time.Second
	probeTimeout  = 10 * time.Second
)

// ProbeDuration returns video/audio duration in seconds using ffprobe.
// Returns 0 if ffprobe fails or file has no duration.
func ProbeDuration(ctx context.Context, path string) int {
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	out, err := exec.CommandContext(probeCtx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		return 0
	}

	var dur float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &dur); err != nil {
		return 0
	}
	return int(dur) + 1 // round up
}

// ExtractAudioChunk extracts a WAV audio chunk from a video file using ffmpeg.
// Output is 16kHz mono PCM suitable for Whisper.
func ExtractAudioChunk(ctx context.Context, videoPath, outputPath string, offsetSec, durationSec int) error {
	ffCtx, cancel := context.WithTimeout(ctx, ffmpegTimeout)
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

// ChunkAndTranscribe splits audio into chunks and transcribes each one.
// Returns nil if transcriber is nil or unavailable.
func ChunkAndTranscribe(ctx context.Context, videoPath, tempDir string, t Transcriber, opts Options) *Transcription {
	if t == nil || !t.Available() {
		return nil
	}

	opts.defaults()
	duration := ProbeDuration(ctx, videoPath)
	if duration <= 0 {
		return nil
	}

	base := filepath.Base(videoPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var chunks []Chunk
	var texts []string

	for offset := 0; offset < duration; offset += opts.ChunkSec {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("%s_%d.wav", name, offset))

		if err := ExtractAudioChunk(ctx, videoPath, chunkPath, offset, opts.ChunkSec); err != nil {
			cleanupFile(chunkPath)
			continue
		}

		result, err := t.Transcribe(ctx, chunkPath)
		cleanupFile(chunkPath)

		if err != nil {
			continue
		}

		text := strings.TrimSpace(result.Text)
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
		return nil
	}

	return &Transcription{
		Text:     strings.Join(texts, " "),
		Duration: float64(duration),
		Chunks:   chunks,
	}
}

func cleanupFile(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}
