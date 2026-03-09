# go-media v0.1.0 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete v0.1.0 MVP — Instagram/Threads video download + transcription as a reusable Go library, extracted from vaelor.

**Architecture:** Pipeline library with pluggable extractors and transcribers. Core orchestrator (`Processor`) wires: extract metadata → download video → chunk audio → transcribe. Instagram extractor uses go-threads client. Transcription via OpenAI-compatible HTTP API (ox-whisper).

**Tech Stack:** Go 1.26, go-threads, go-stt, ffmpeg/ffprobe (exec), golangci-lint v2

**Current State:** Scaffolding complete — 14 Go files, 13 passing tests, 0 lint issues. Core types, interfaces, registry, download, audio chunking, processor, Instagram extractor, and OpenAI/go-stt transcriber backends are written. Missing: tests for transcriber backends, audio module tests, integration test, CLAUDE.md, git setup, vaelor migration.

---

### Task 1: OpenAI Transcriber Tests

**Files:**
- Create: `transcribe/openai/transcriber_test.go`

**Step 1: Write tests**

```go
package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscribeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/transcriptions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct == "" {
			t.Fatal("missing Content-Type")
		}

		resp := map[string]any{
			"text":     "hello world",
			"language": "en",
			"duration": 2.5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	tr := New(srv.URL)

	// Create a dummy audio file
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "test.wav")
	os.WriteFile(audioPath, []byte("fake-audio"), 0o644)

	result, err := tr.Transcribe(context.Background(), audioPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result.Text)
	}
	if result.Language != "en" {
		t.Fatalf("expected 'en', got %q", result.Language)
	}
}

func TestTranscribeAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	tr := New(srv.URL)
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "test.wav")
	os.WriteFile(audioPath, []byte("fake"), 0o644)

	_, err := tr.Transcribe(context.Background(), audioPath)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestTranscribeWithAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		resp := map[string]any{"text": "ok"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	tr := New(srv.URL, WithAPIKey("sk-test-123"))
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "test.wav")
	os.WriteFile(audioPath, []byte("fake"), 0o644)

	tr.Transcribe(context.Background(), audioPath)
	if gotAuth != "Bearer sk-test-123" {
		t.Fatalf("expected Bearer auth, got %q", gotAuth)
	}
}

func TestAvailableTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer srv.Close()

	tr := New(srv.URL)
	if !tr.Available() {
		t.Fatal("expected Available() = true")
	}
}

func TestAvailableFalse(t *testing.T) {
	tr := New("")
	if tr.Available() {
		t.Fatal("expected Available() = false for empty URL")
	}
}
```

**Step 2: Run tests**

Run: `GOWORK=off go test -v ./transcribe/openai/...`
Expected: PASS (all 5 tests)

**Step 3: Commit**

```bash
git add transcribe/openai/transcriber_test.go
git commit -m "test: add OpenAI transcriber tests"
```

---

### Task 2: Audio Module Tests

**Files:**
- Create: `audio_test.go`

**Step 1: Write tests**

```go
package media_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anatolykoptev/go-media"
)

func hasFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func hasFFprobe() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

func TestProbeDurationNoFile(t *testing.T) {
	dur := media.ProbeDuration(context.Background(), "/nonexistent/file.mp4")
	if dur != 0 {
		t.Fatalf("expected 0 for nonexistent file, got %d", dur)
	}
}

func TestExtractAudioChunkNoFFmpeg(t *testing.T) {
	if hasFFmpeg() {
		t.Skip("ffmpeg is available, skipping negative test")
	}
	err := media.ExtractAudioChunk(context.Background(), "input.mp4", "output.wav", 0, 20)
	if err == nil {
		t.Fatal("expected error when ffmpeg not available")
	}
}

func TestChunkAndTranscribeNilTranscriber(t *testing.T) {
	result := media.ChunkAndTranscribe(context.Background(), "video.mp4", t.TempDir(), nil, media.Options{})
	if result != nil {
		t.Fatal("expected nil for nil transcriber")
	}
}

type unavailableTranscriber struct{}

func (u *unavailableTranscriber) Transcribe(_ context.Context, _ string) (*media.Transcription, error) {
	return nil, nil
}
func (u *unavailableTranscriber) Available() bool { return false }

func TestChunkAndTranscribeUnavailable(t *testing.T) {
	result := media.ChunkAndTranscribe(context.Background(), "video.mp4", t.TempDir(), &unavailableTranscriber{}, media.Options{})
	if result != nil {
		t.Fatal("expected nil for unavailable transcriber")
	}
}

// Integration test: only runs if ffmpeg + ffprobe available
func TestExtractAudioChunkIntegration(t *testing.T) {
	if !hasFFmpeg() || !hasFFprobe() {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	// Generate a 3-second silent video with ffmpeg
	videoPath := filepath.Join(dir, "test.mp4")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono",
		"-f", "lavfi", "-i", "color=c=black:s=320x240:r=1",
		"-t", "3", "-y", videoPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to generate test video: %v", err)
	}

	// Probe duration
	dur := media.ProbeDuration(context.Background(), videoPath)
	if dur < 3 || dur > 5 {
		t.Fatalf("expected duration ~3-4, got %d", dur)
	}

	// Extract audio chunk
	audioPath := filepath.Join(dir, "chunk.wav")
	err := media.ExtractAudioChunk(context.Background(), videoPath, audioPath, 0, 2)
	if err != nil {
		t.Fatalf("extract chunk: %v", err)
	}

	fi, err := os.Stat(audioPath)
	if err != nil {
		t.Fatalf("stat chunk: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatal("extracted chunk is empty")
	}
}
```

**Step 2: Run tests**

Run: `GOWORK=off go test -v -run TestProbeDuration,TestChunkAndTranscribe,TestExtractAudioChunk .`
Expected: PASS (tests skip gracefully if ffmpeg not available)

**Step 3: Commit**

```bash
git add audio_test.go
git commit -m "test: add audio module tests with ffmpeg integration"
```

---

### Task 3: CLAUDE.md + README

**Files:**
- Create: `CLAUDE.md`
- Create: `README.md`

**Step 1: Write CLAUDE.md**

```markdown
# go-media

**Module**: `github.com/anatolykoptev/go-media`
**Repo**: `/home/krolik/src/go-media/`

## Architecture

Pipeline library: Extract → Download → Audio → Transcribe.
- `Extractor` interface: platform-specific video URL extraction
- `Transcriber` interface: audio-to-text conversion
- `Processor`: orchestrates the full pipeline

## Commands

```bash
make test    # run tests
make lint    # golangci-lint
make build   # build all packages
```

## Rules

- Source files ≤ 200 lines
- No logging — return errors
- No init() magic — explicit registration
- exec.Command for ffmpeg/ffprobe (no cgo)
- Tests skip gracefully if ffmpeg not in PATH
```

**Step 2: Write README.md**

Short README with: description, install, usage example, supported platforms, license.

**Step 3: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: add CLAUDE.md and README"
```

---

### Task 4: GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`

**Step 1: Write CI config**

Standard CI: lint + test matrix. Include ffmpeg install step for audio tests.

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: golangci/golangci-lint-action@v7
        with:
          version: latest
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: sudo apt-get update && sudo apt-get install -y ffmpeg
      - run: go test -race -coverprofile=coverage.txt ./...
```

**Step 2: Commit**

```bash
mkdir -p .github/workflows
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions workflow"
```

---

### Task 5: Initial Git Commit + Push

**Step 1: Stage all files**

```bash
git add -A
git status  # verify no secrets, no .env files
```

**Step 2: Commit**

```bash
git commit -m "feat: go-media v0.1.0 scaffold

Universal media processing library: download video + extract audio + transcribe.
- Core types, interfaces, registry, processor
- Instagram/Threads extractor (go-threads)
- OpenAI-compatible transcriber (ox-whisper/Groq)
- go-stt transcriber wrapper
- HTTP downloader with size limits
- FFmpeg audio extraction + chunked transcription
- 13+ tests, 0 lint issues
- golangci-lint v2 + pre-commit + Makefile"
```

**Step 3: Push**

```bash
git remote add origin https://github.com/anatolykoptev/go-media.git
git push -u origin main
```

**Step 4: Tag**

```bash
git tag v0.1.0
git push origin v0.1.0
```

---

### Task 6: Migrate Vaelor to Use go-media

**Files:**
- Modify: `/home/krolik/src/vaelor/pkg/tools/instagram.go`
- Modify: `/home/krolik/src/vaelor/pkg/tools/instagram_media.go`
- Modify: `/home/krolik/src/vaelor/pkg/tools/instagram_transcribe.go`
- Modify: `/home/krolik/src/vaelor/go.mod`

**Step 1: Add go-media dependency to vaelor**

```bash
cd /home/krolik/src/vaelor
go get github.com/anatolykoptev/go-media@v0.1.0
```

**Step 2: Refactor instagram.go to use go-media**

Replace internal download/transcribe with `media.Processor`. Keep:
- `InstagramTool` struct (vaelor tool interface)
- `MediaSender` (Telegram delivery)
- `formatToolResult()` / `formatVideoCaption()`

The tool's `Execute()` method should:
1. Use `media.Processor.Process()` for the pipeline
2. Keep Telegram-specific formatting and media sending

**Step 3: Remove duplicated code**

Delete from vaelor:
- `parseInstagramURL()` → now in `go-media/extract/instagram`
- `downloadVideo()` → now in `go-media`
- `transcribeVideo()` → now in `go-media`
- `getAudioDuration()` → now in `go-media`

Keep in vaelor:
- `formatVideoCaption()` (Telegram caption limit)
- `formatToolResult()` (LLM instruction injection)
- `looksLikeRawSTT()` (vaelor-specific)

**Step 4: Run vaelor tests**

```bash
cd /home/krolik/src/vaelor
go test ./...
```

**Step 5: Commit in vaelor**

```bash
git add -A
git commit -m "refactor: use go-media for Instagram video processing

Replace internal download/transcribe with go-media library.
Keeps Telegram-specific formatting and media delivery."
```

**Step 6: Deploy vaelor**

```bash
cd /home/krolik/src/vaelor
make deploy
```

---

## Summary

| Task | What | Tests | Files |
|------|------|-------|-------|
| 1 | OpenAI transcriber tests | 5 new | 1 |
| 2 | Audio module tests | 5 new | 1 |
| 3 | CLAUDE.md + README | — | 2 |
| 4 | GitHub Actions CI | — | 1 |
| 5 | Git commit + push + tag | — | — |
| 6 | Vaelor migration | existing | 4 modified |

**Estimated new tests:** 10
**Total after completion:** ~23 tests
