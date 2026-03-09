package media_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anatolykoptev/go-media"
)

type mockHTTPClient struct {
	statusCode int
	body       string
	contentLen int64
	err        error
}

func (m *mockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	resp := &http.Response{
		StatusCode:    m.statusCode,
		ContentLength: m.contentLen,
		Body:          io.NopCloser(strings.NewReader(m.body)),
	}
	return resp, nil
}

func TestDownloadFileSuccess(t *testing.T) {
	client := &mockHTTPClient{
		statusCode: http.StatusOK,
		body:       "video-content-here",
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "test.mp4")

	err := media.DownloadFile(context.Background(), client, "https://cdn.example.com/video.mp4", dest, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "video-content-here" {
		t.Fatalf("unexpected content: %s", string(data))
	}
}

func TestDownloadFileHTTPError(t *testing.T) {
	client := &mockHTTPClient{statusCode: http.StatusForbidden}

	dir := t.TempDir()
	dest := filepath.Join(dir, "test.mp4")

	err := media.DownloadFile(context.Background(), client, "https://cdn.example.com/video.mp4", dest, 0)
	if err == nil {
		t.Fatal("expected error for 403 status")
	}
}

func TestDownloadFileContentLengthExceeded(t *testing.T) {
	client := &mockHTTPClient{
		statusCode: http.StatusOK,
		contentLen: 100 * 1024 * 1024, // 100 MB
		body:       "data",
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "test.mp4")

	err := media.DownloadFile(context.Background(), client, "https://cdn.example.com/video.mp4", dest, 50*1024*1024)
	if err == nil {
		t.Fatal("expected error for oversized content")
	}
}

func TestDownloadFileBodyOverflow(t *testing.T) {
	bigBody := strings.Repeat("x", 1024)
	client := &mockHTTPClient{
		statusCode: http.StatusOK,
		body:       bigBody,
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "test.mp4")

	err := media.DownloadFile(context.Background(), client, "https://cdn.example.com/video.mp4", dest, 512)
	if err == nil {
		t.Fatal("expected error for body overflow")
	}

	// File should be cleaned up
	if _, err := os.Stat(dest); err == nil {
		t.Fatal("expected file to be cleaned up after overflow")
	}
}
