package media

import (
	"context"
	"fmt"
)

// Extractor fetches media metadata from a platform-specific URL.
type Extractor interface {
	// Name returns the platform name (e.g. "instagram", "youtube").
	Name() string
	// Match returns true if the URL belongs to this platform.
	Match(url string) bool
	// Extract fetches media metadata including the direct video URL.
	Extract(ctx context.Context, url string) (*Media, error)
}

// BudgetAwareExtractor is an optional capability implemented by extractors that
// can pick a quality representation fitting a byte budget (e.g. choosing a DASH
// representation, or passing --max-filesize to yt-dlp). The Registry routes to
// ExtractWithBudget when the extractor implements it; otherwise it falls back
// to Extract and the budget is enforced later by DownloadFile. maxSize == 0
// means no limit.
type BudgetAwareExtractor interface {
	ExtractWithBudget(ctx context.Context, url string, maxSize int64) (*Media, error)
}

// Registry holds registered extractors and dispatches URLs to the matching one.
type Registry struct {
	extractors []Extractor
}

// NewRegistry creates an empty extractor registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds an extractor to the registry.
func (r *Registry) Register(e Extractor) {
	r.extractors = append(r.extractors, e)
}

// Match finds the first extractor that matches the given URL.
// Returns nil if no extractor matches.
func (r *Registry) Match(url string) Extractor {
	for _, e := range r.extractors {
		if e.Match(url) {
			return e
		}
	}
	return nil
}

// Extract finds the matching extractor and extracts media metadata.
func (r *Registry) Extract(ctx context.Context, url string) (*Media, error) {
	e := r.Match(url)
	if e == nil {
		return nil, fmt.Errorf("no extractor matches URL: %s", url)
	}
	return e.Extract(ctx, url)
}

// ExtractWithBudget is like Extract but threads a byte budget into extractors
// that implement BudgetAwareExtractor. Plain extractors fall back to Extract
// (the budget is still enforced later by DownloadFile for URL-based downloads).
func (r *Registry) ExtractWithBudget(ctx context.Context, url string, maxSize int64) (*Media, error) {
	e := r.Match(url)
	if e == nil {
		return nil, fmt.Errorf("no extractor matches URL: %s", url)
	}
	if bx, ok := e.(BudgetAwareExtractor); ok {
		return bx.ExtractWithBudget(ctx, url, maxSize)
	}
	return e.Extract(ctx, url)
}

// Platforms returns the names of all registered extractors.
func (r *Registry) Platforms() []string {
	names := make([]string, len(r.extractors))
	for i, e := range r.extractors {
		names[i] = e.Name()
	}
	return names
}
