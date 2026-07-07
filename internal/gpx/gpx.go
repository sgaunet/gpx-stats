// Package gpx parses GPX files into an in-memory Track.
//
// Parsing uses the standard library encoding/xml, which ignores document type
// definitions (DTDs) and never resolves external entities. As a result,
// external-entity (XXE) and entity-expansion ("billion laughs") attacks cannot
// be triggered: a reference to a declared custom entity fails to resolve and
// decoding returns an error. Input is additionally bounded by a maximum byte
// size and a maximum number of track points.
package gpx

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// TrackPoint is a single recorded sample. Elevation and time are optional and
// signalled by the Has* flags.
type TrackPoint struct {
	Lat     float64
	Lon     float64
	Ele     float64
	HasEle  bool
	Time    time.Time
	HasTime bool
}

// Track is a parsed activity: track points aggregated across all segments, plus
// flags indicating whether elevation and timestamps are available throughout.
type Track struct {
	Points       []TrackPoint
	HasElevation bool
	HasTimes     bool
}

// xmlTrkpt mirrors a <trkpt> element for decoding.
type xmlTrkpt struct {
	Lat  float64  `xml:"lat,attr"`
	Lon  float64  `xml:"lon,attr"`
	Ele  *float64 `xml:"ele"`
	Time *string  `xml:"time"`
}

// Parse reads a GPX document from r and returns the aggregated Track.
//
// maxBytes bounds the number of bytes read (protecting against memory
// exhaustion); maxPoints bounds the number of track points accepted. Exceeding
// either limit, or malformed / non-GPX input, returns an actionable error.
func Parse(r io.Reader, maxBytes int64, maxPoints int) (Track, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return Track{}, fmt.Errorf("reading gpx input: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return Track{}, fmt.Errorf("gpx input exceeds the maximum size of %d bytes", maxBytes)
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	// Intentionally leave dec.Entity and dec.CharsetReader unset: this keeps
	// encoding/xml's safe default of not resolving custom or external entities.

	var track Track
	sawGPX := false

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Track{}, fmt.Errorf("parsing gpx: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "gpx":
			sawGPX = true
		case "trkpt":
			if len(track.Points) >= maxPoints {
				return Track{}, fmt.Errorf("gpx exceeds the maximum of %d track points", maxPoints)
			}
			var xp xmlTrkpt
			if err := dec.DecodeElement(&xp, &se); err != nil {
				return Track{}, fmt.Errorf("parsing track point: %w", err)
			}
			pt, err := xp.toPoint()
			if err != nil {
				return Track{}, err
			}
			track.Points = append(track.Points, pt)
		}
	}

	if !sawGPX {
		return Track{}, errors.New("not a GPX file: missing <gpx> root element")
	}
	track.finalize()
	return track, nil
}

// toPoint validates and converts a decoded <trkpt> into a TrackPoint.
func (x xmlTrkpt) toPoint() (TrackPoint, error) {
	if x.Lat < -90 || x.Lat > 90 {
		return TrackPoint{}, fmt.Errorf("invalid latitude %g (must be between -90 and 90)", x.Lat)
	}
	if x.Lon < -180 || x.Lon > 180 {
		return TrackPoint{}, fmt.Errorf("invalid longitude %g (must be between -180 and 180)", x.Lon)
	}
	pt := TrackPoint{Lat: x.Lat, Lon: x.Lon}
	if x.Ele != nil {
		pt.Ele = *x.Ele
		pt.HasEle = true
	}
	if x.Time != nil {
		t, err := parseTime(*x.Time)
		if err != nil {
			return TrackPoint{}, fmt.Errorf("invalid track point time %q: %w", *x.Time, err)
		}
		pt.Time = t
		pt.HasTime = true
	}
	return pt, nil
}

// parseTime accepts RFC3339 timestamps, with or without fractional seconds.
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("expected an RFC3339 timestamp")
}

// finalize sets the track-level availability flags. Elevation and timestamps
// are considered available only when at least two points carry them (enough to
// compute gain and elapsed time) and no point is missing them.
func (t *Track) finalize() {
	if len(t.Points) < 2 {
		t.HasElevation = false
		t.HasTimes = false
		return
	}
	hasEle, hasTime := true, true
	for _, p := range t.Points {
		if !p.HasEle {
			hasEle = false
		}
		if !p.HasTime {
			hasTime = false
		}
	}
	t.HasElevation = hasEle
	t.HasTimes = hasTime
}
