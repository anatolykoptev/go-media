package openai

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscribeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/transcriptions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"hello world","language":"en","duration":2.5}`))
	}))
	defer srv.Close()

	audio := writeDummyAudio(t)
	tr := New(srv.URL)

	result, err := tr.Transcribe(t.Context(), audio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "hello world" {
		t.Errorf("text = %q, want %q", result.Text, "hello world")
	}
	if result.Language != "en" {
		t.Errorf("language = %q, want %q", result.Language, "en")
	}
	if result.Duration != 2.5 {
		t.Errorf("duration = %f, want %f", result.Duration, 2.5)
	}
}

func TestTranscribeAPIError(t *testing.T) {
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
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status 500: %v", err)
	}
}

func TestTranscribeWithAPIKey(t *testing.T) {
	const apiKey = "test-secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		want := "Bearer " + apiKey
		if auth != want {
			t.Errorf("Authorization = %q, want %q", auth, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"ok","language":"en","duration":1.0}`))
	}))
	defer srv.Close()

	audio := writeDummyAudio(t)
	tr := New(srv.URL, WithAPIKey(apiKey))

	_, err := tr.Transcribe(t.Context(), audio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAvailableTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
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
	tr := New("")
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
