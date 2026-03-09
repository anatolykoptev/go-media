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

// Platforms returns the names of all registered extractors.
func (r *Registry) Platforms() []string {
	names := make([]string, len(r.extractors))
	for i, e := range r.extractors {
		names[i] = e.Name()
	}
	return names
}
