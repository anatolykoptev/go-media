// Package openai provides a Transcriber that uses an OpenAI-compatible STT API
// (works with ox-whisper, Groq, OpenAI, etc.).
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/anatolykoptev/go-media"
)

const (
	defaultTimeout = 60 * time.Second
	healthTimeout  = 5 * time.Second
)

// Transcriber calls an OpenAI-compatible /v1/audio/transcriptions endpoint.
type Transcriber struct {
	baseURL    string
	apiKey     string
	model      string
	language   string
	httpClient *http.Client
}

// Option configures the Transcriber.
type Option func(*Transcriber)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(t *Transcriber) { t.apiKey = key }
}

// WithModel sets the Whisper model name (default: "whisper-large-v3").
func WithModel(model string) Option {
	return func(t *Transcriber) { t.model = model }
}

// WithLanguage sets the language hint for transcription.
func WithLanguage(lang string) Option {
	return func(t *Transcriber) { t.language = lang }
}

// New creates a Transcriber for the given OpenAI-compatible base URL.
// Example: New("http://localhost:8092/v1")
func New(baseURL string, opts ...Option) *Transcriber {
	t := &Transcriber{
		baseURL: baseURL,
		model:   "whisper-large-v3",
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

func (t *Transcriber) Transcribe(ctx context.Context, audioPath string) (*media.Transcription, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only file

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	part, err := w.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("create form: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy audio: %w", err)
	}

	_ = w.WriteField("model", t.model)
	_ = w.WriteField("response_format", "verbose_json")
	if t.language != "" {
		_ = w.WriteField("language", t.language)
	}
	_ = w.Close()

	reqURL := t.baseURL + "/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}

	resp, err := t.httpClient.Do(req) //nolint:gosec // URL from trusted config
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Text     string  `json:"text"`
		Language string  `json:"language"`
		Duration float64 `json:"duration"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &media.Transcription{
		Text:     result.Text,
		Language: result.Language,
		Duration: result.Duration,
	}, nil
}

func (t *Transcriber) Available() bool {
	if t.baseURL == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	modelsURL := t.baseURL + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return false
	}
	resp, err := t.httpClient.Do(req) //nolint:gosec // URL from trusted config
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
