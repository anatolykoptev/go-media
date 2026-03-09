package media

import "context"

// Transcriber converts an audio file to text.
type Transcriber interface {
	// Transcribe processes an audio file and returns the transcription.
	// audioPath must be a valid file path to a WAV/MP3/OGG file.
	Transcribe(ctx context.Context, audioPath string) (*Transcription, error)
	// Available reports whether the transcription backend is reachable.
	Available() bool
}
