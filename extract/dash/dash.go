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
	ID        string `xml:"id,attr"`
	Width     int    `xml:"width,attr"`
	Height    int    `xml:"height,attr"`
	Bandwidth int64  `xml:"bandwidth,attr"`
	Codecs    string `xml:"codecs,attr"`
	Label     string `xml:"qualityRanking,attr"` // best-effort label slot
	BaseURL   string `xml:"BaseURL"`
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
		mime := as.MimeType
		if mime == "" {
			mime = as.ContentType
		}
		isVideo := strings.HasPrefix(mime, "video/")
		isAudio := strings.HasPrefix(mime, "audio/")
		if !isVideo && !isAudio {
			continue
		}
		for _, rx := range as.Representations {
			rep := Representation{
				ID:        rx.ID,
				MimeType:  mime,
				Width:     rx.Width,
				Height:    rx.Height,
				Bandwidth: rx.Bandwidth,
				Codecs:    rx.Codecs,
				Label:     rx.Label,
				URL:       strings.TrimSpace(rx.BaseURL),
			}
			if isVideo {
				man.Videos = append(man.Videos, rep)
			} else {
				man.Audios = append(man.Audios, rep)
			}
		}
	}

	return man, nil
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
