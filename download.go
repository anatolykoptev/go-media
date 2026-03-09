package media

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultDownloadTimeout = 120 * time.Second

// HTTPDoer abstracts an HTTP client for testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DownloadFile downloads a URL to a local file with timeout and size limit.
func DownloadFile(ctx context.Context, client HTTPDoer, url, destPath string, maxSize int64) error {
	dlCtx, cancel := context.WithTimeout(ctx, defaultDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Check Content-Length before downloading
	if maxSize > 0 && resp.ContentLength > maxSize {
		return fmt.Errorf("file too large: %d bytes (limit %d)", resp.ContentLength, maxSize)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close() //nolint:errcheck // temp file

	var reader io.Reader = resp.Body
	if maxSize > 0 {
		reader = io.LimitReader(resp.Body, maxSize+1) // +1 to detect overflow
	}

	written, err := io.Copy(f, reader)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if maxSize > 0 && written > maxSize {
		_ = os.Remove(destPath)
		return fmt.Errorf("file too large: %d bytes (limit %d)", written, maxSize)
	}

	return nil
}
