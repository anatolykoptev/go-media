package gostt

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stt "github.com/anatolykoptev/go-stt"
)

func TestTranscribeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"hello","language":"en","duration":5.0}`))
	}))
	defer srv.Close()

	audio := writeDummyAudio(t)
	tr := New(srv.URL)

	result, err := tr.Transcribe(t.Context(), audio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "hello" {
		t.Errorf("text = %q, want %q", result.Text, "hello")
	}
	if result.Language != "en" {
		t.Errorf("language = %q, want %q", result.Language, "en")
	}
	if result.Duration != 5.0 {
		t.Errorf("duration = %f, want %f", result.Duration, 5.0)
	}
}

func TestTranscribeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	audio := writeDummyAudio(t)
	tr := New(srv.URL)

	_, err := tr.Transcribe(t.Context(), audio)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "go-stt transcribe") {
		t.Errorf("error should be wrapped with 'go-stt transcribe': %v", err)
	}
}

func TestAvailableTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := New(srv.URL)
	if !tr.Available() {
		t.Error("Available() = false, want true")
	}
}

func TestAvailableFalse(t *testing.T) {
	tr := New("http://127.0.0.1:1", stt.WithTimeout(100*time.Millisecond))
	if tr.Available() {
		t.Error("Available() = true, want false")
	}
}

func writeDummyAudio(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.wav")
	if err := os.WriteFile(p, []byte("fake-audio-data"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}
