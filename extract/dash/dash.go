// Package dash parses MPEG-DASH MPD manifests into video/audio representations
// and selects a representation pair that fits a byte budget. Used by extractors
// (e.g. Instagram) that receive a raw MPD string carrying higher-than-720p
// renditions not exposed in the platform's video_versions list.
package dash

import (
	"encoding/xml"
	"fmt"
	"strings"
)

const (
	secondsPerHour = 3600
	secondsPerMin  = 60
	bitsPerByte    = 8

	kindVideo = "video"
	kindAudio = "audio"
)

// Representation is a single DASH representation (video or audio) with the
// fields needed for quality selection and download. URL is the BaseURL
// chardata; for Instagram manifests these are absolute CDN URLs.
type Representation struct {
	ID        string
	MimeType  string
	Width     int
	Height    int
	Bandwidth int64 // bits per second
	Codecs    string
	Label     string // quality label if present in the manifest, else empty
	URL       string // resolved media URL from BaseURL
	// ContentLength is the exact download size in bytes when the manifest
	// carries it (e.g. Instagram's FBContentLength attribute). 0 when absent;
	// the selector then falls back to the bandwidth*duration/8 estimate.
	ContentLength int64
}

// Manifest is a parsed MPD split into video and audio AdaptationSets.
type Manifest struct {
	Duration float64 // seconds, from mediaPresentationDuration
	Videos   []Representation
	Audios   []Representation
}

// mpdXML mirrors the subset of the MPD schema we read. Only the fields used for
// selection are decoded; unknown elements/attributes are ignored.
type mpdXML struct {
	XMLName  xml.Name    `xml:"MPD"`
	Duration string      `xml:"mediaPresentationDuration,attr"`
	Periods  []periodXML `xml:"Period"`
	// Some manifests put AdaptationSets directly under MPD (no Period).
	AdaptationSets []adaptationXML `xml:"AdaptationSet"`
}

type periodXML struct {
	AdaptationSets []adaptationXML `xml:"AdaptationSet"`
}

type adaptationXML struct {
	MimeType        string              `xml:"mimeType,attr"`
	ContentType     string              `xml:"contentType,attr"`
	Representations []representationXML `xml:"Representation"`
}

type representationXML struct {
	ID              string `xml:"id,attr"`
	Width           int    `xml:"width,attr"`
	Height          int    `xml:"height,attr"`
	Bandwidth       int64  `xml:"bandwidth,attr"`
	Codecs          string `xml:"codecs,attr"`
	MimeType        string `xml:"mimeType,attr"`
	FBContentLength int64  `xml:"FBContentLength,attr"`
	BaseURL         string `xml:"BaseURL"`
}

// ParseManifest parses a raw MPD XML string into a Manifest. A malformed or
// unexpected manifest returns an error; it never panics.
func ParseManifest(raw string) (*Manifest, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("dash: empty manifest")
	}

	var doc mpdXML
	if err := xml.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, fmt.Errorf("dash: parse mpd: %w", err)
	}

	man := &Manifest{}
	if doc.Duration != "" {
		d, err := parseDuration(doc.Duration)
		if err != nil {
			return nil, fmt.Errorf("dash: parse duration: %w", err)
		}
		man.Duration = d
	}

	// Collect AdaptationSets from both MPD-level and Period-level.
	sets := append([]adaptationXML(nil), doc.AdaptationSets...)
	for _, p := range doc.Periods {
		sets = append(sets, p.AdaptationSets...)
	}

	for _, as := range sets {
		kind := adaptationKind(as)
		if kind == "" {
			continue
		}
		for _, rx := range as.Representations {
			rep := buildRepresentation(as, rx)
			if kind == kindVideo {
				man.Videos = append(man.Videos, rep)
			} else {
				man.Audios = append(man.Audios, rep)
			}
		}
	}

	return man, nil
}

// buildRepresentation maps a parsed Representation element onto the public
// Representation type. The per-rep MimeType is resolved from the most specific
// source available: AdaptationSet.mimeType (full, e.g. "video/mp4") is
// preferred; else the Representation-level mimeType (Instagram puts it here);
// else the bare AdaptationSet.contentType ("video") as a last resort.
func buildRepresentation(as adaptationXML, rx representationXML) Representation {
	repMime := strings.TrimSpace(as.MimeType)
	if repMime == "" {
		repMime = strings.TrimSpace(rx.MimeType)
	}
	if repMime == "" {
		repMime = strings.TrimSpace(as.ContentType)
	}
	return Representation{
		ID:            rx.ID,
		MimeType:      repMime,
		Width:         rx.Width,
		Height:        rx.Height,
		Bandwidth:     rx.Bandwidth,
		Codecs:        rx.Codecs,
		URL:           strings.TrimSpace(rx.BaseURL),
		ContentLength: rx.FBContentLength,
	}
}

// adaptationKind classifies an AdaptationSet as "video" or "audio" ("" if
// neither). A set is video/audio when its mimeType is "video/mp4"/"audio/mp4"
// OR its bare contentType is "video"/"audio" (no slash — Instagram sends this
// form, with mimeType on the Representation instead). When the AdaptationSet
// carries neither, the Representation-level mimeType is consulted (that is
// where Instagram puts it). Matching is case-insensitive and tolerates
// surrounding whitespace.
func adaptationKind(as adaptationXML) string {
	candidate := strings.TrimSpace(as.MimeType)
	if candidate == "" {
		candidate = strings.TrimSpace(as.ContentType)
	}
	if candidate == "" {
		for _, rx := range as.Representations {
			if rm := strings.TrimSpace(rx.MimeType); rm != "" {
				candidate = rm
				break
			}
		}
	}
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	switch {
	case candidate == kindVideo || strings.HasPrefix(candidate, kindVideo+"/"):
		return kindVideo
	case candidate == kindAudio || strings.HasPrefix(candidate, kindAudio+"/"):
		return kindAudio
	}
	return ""
}

// parseDuration parses an ISO 8601 PT duration string (e.g. "PT0H0M30.000S")
// into seconds.
func parseDuration(s string) (float64, error) {
	if !strings.HasPrefix(s, "PT") {
		return 0, fmt.Errorf("dash: duration %q missing PT prefix", s)
	}
	body := strings.TrimPrefix(s, "PT")
	var seconds float64
	var num strings.Builder
	var unit byte
	flush := func() error {
		if num.Len() == 0 {
			return nil
		}
		val := 0.0
		_, err := fmt.Sscanf(num.String(), "%f", &val)
		if err != nil {
			return fmt.Errorf("dash: bad duration number %q", num.String())
		}
		switch unit {
		case 'H':
			seconds += val * secondsPerHour
		case 'M':
			seconds += val * secondsPerMin
		case 'S':
			seconds += val
		default:
			return fmt.Errorf("dash: unknown duration unit %q", unit)
		}
		num.Reset()
		return nil
	}
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch c {
		case 'H', 'M', 'S':
			unit = c
			if err := flush(); err != nil {
				return 0, err
			}
		default:
			num.WriteByte(c)
		}
	}
	if num.Len() > 0 {
		return 0, fmt.Errorf("dash: trailing data in duration %q", s)
	}
	return seconds, nil
}

// EstimatedSize returns the estimated download size in bytes for a
// representation: bandwidth(bits/s) * duration(s) / bitsPerByte. Returns 0 if
// the manifest carried no duration.
func EstimatedSize(r Representation, duration float64) int64 {
	if duration <= 0 || r.Bandwidth <= 0 {
		return 0
	}
	return int64(float64(r.Bandwidth) * duration / bitsPerByte)
}
