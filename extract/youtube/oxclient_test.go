package youtube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const (
	testOxVideoURL = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	testOxHeader   = "X-Internal-Secret"
	testOxSecret   = "s3cret"
)

// headerTransport injects a header on every request, standing in for a
// go-kit svcauth transport.
type headerTransport struct {
	base       http.RoundTripper
	key, value string
}

func (t headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set(t.key, t.value)
	return t.base.RoundTrip(r)
}

// oxStandIn serves /media/download and records the header seen on each call.
func oxStandIn(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/media/download" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		seen = append(seen, r.Header.Get(testOxHeader))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"platform":"youtube","title":"t","files":[{"path":"/tmp/x.mp4","size_bytes":1,"width":1,"height":1}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// ox-only extractor: kkdai disabled, no yt-dlp, so Extract reaches tier 3.
func TestExtract_OxHTTPClient_Injected(t *testing.T) {
	srv, seen := oxStandIn(t)
	cl := &http.Client{Transport: headerTransport{base: http.DefaultTransport, key: testOxHeader, value: testOxSecret}}
	e := New(Config{DisableKkdai: true, OxBrowserURL: srv.URL, OxHTTPClient: cl})

	if _, err := e.Extract(context.Background(), testOxVideoURL); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	got := seen()
	if len(got) != 1 || got[0] != testOxSecret {
		t.Fatalf("ox-browser saw %s=%q, want exactly one call with %q", testOxHeader, got, testOxSecret)
	}
}

func TestExtract_OxHTTPClient_NilKeepsDefault(t *testing.T) {
	srv, seen := oxStandIn(t)
	e := New(Config{DisableKkdai: true, OxBrowserURL: srv.URL})

	if _, err := e.Extract(context.Background(), testOxVideoURL); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	got := seen()
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("ox-browser saw %s=%q, want exactly one call with no header", testOxHeader, got)
	}
}
