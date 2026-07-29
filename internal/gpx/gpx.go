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

	// SegmentIndex identifies the <trkseg> this point was recorded in, counting
	// from zero across the whole document. Its zero value means "the only
	// segment", so a track built by hand without setting it behaves exactly like
	// a single-segment recording.
	SegmentIndex int
}

// SameSegment reports whether two track points were recorded in the same
// <trkseg>.
//
// A segment boundary means recording was interrupted: the device was stopped,
// possibly moved, and started again. No distance-derived metric may accrue
// between two points on opposite sides of one, because the straight line
// joining them is not a path that was travelled.
func SameSegment(a, b TrackPoint) bool { return a.SegmentIndex == b.SegmentIndex }

// Track is a parsed activity: track points from every segment in document
// order, each tagged with the segment it came from, plus flags indicating
// whether elevation and timestamps are available throughout.
type Track struct {
	Points       []TrackPoint
	HasElevation bool
	HasTimes     bool

	// Creator is the <gpx creator="..."> attribute: the device or application
	// that wrote the file. Empty when the attribute is absent.
	Creator string
	// Name and Type come from the first <trk> that carries a non-empty one. A
	// file holding several tracks is analysed as one activity, so one name is
	// reported rather than a joined list. Empty when absent.
	Name string
	Type string
	// MetadataTime is <metadata><time>: when the FILE was written. Many tools
	// set it to the export time, so it is not the activity's start — that is
	// derived from the track points themselves.
	MetadataTime    time.Time
	HasMetadataTime bool
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

	// path holds the element names from the root down, so elements can be
	// matched by their position rather than by name alone. <name> appears under
	// <metadata>, <trk>, <wpt> and <rte> alike, and only the <trk> one is the
	// activity's name.
	var path []string

	// Segment tracking is lazy: a <trkseg> only arms a pending boundary, and the
	// next track point spends it. That keeps the indices dense — an empty
	// <trkseg> between two populated ones burns no index — and stops the opening
	// <trkseg> of a document from pushing the very first point into segment 1.
	segIndex := 0
	pendingSegment := false
	sawPoint := false

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Track{}, fmt.Errorf("parsing gpx: %w", err)
		}
		if _, ok := tok.(xml.EndElement); ok {
			// dec.Strict guarantees balanced tokens, so this only ever pops an
			// element this loop pushed. The guard costs nothing and removes the
			// possibility of a panic on a decoder change.
			if len(path) > 0 {
				path = path[:len(path)-1]
			}
			continue
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		path = append(path, se.Name.Local)

		// consumed records whether DecodeElement swallowed this element whole.
		// When it did, no matching EndElement will arrive, so the element has to
		// be popped here instead.
		consumed := false

		switch se.Name.Local {
		case "gpx":
			sawGPX = true
			for _, a := range se.Attr {
				if a.Name.Local == "creator" {
					track.Creator = cleanIdentity(a.Value)
				}
			}
		case "name", "type":
			// Only under <trk>: the identically named elements under <metadata>,
			// <wpt> and <rte> describe something else entirely.
			if !parentIs(path, "trk") {
				break
			}
			s, derr := decodeString(dec, &se)
			if derr != nil {
				return Track{}, derr
			}
			consumed = true
			// The first non-empty one wins, so a second <trk> cannot rename an
			// activity the first already named.
			if s != "" {
				if se.Name.Local == "name" && track.Name == "" {
					track.Name = s
				} else if se.Name.Local == "type" && track.Type == "" {
					track.Type = s
				}
			}
		case "time":
			if !parentIs(path, "metadata") {
				break
			}
			s, derr := decodeString(dec, &se)
			if derr != nil {
				return Track{}, derr
			}
			consumed = true
			// A malformed metadata time is descriptive, not structural: it must
			// not cost the user the whole analysis.
			if t, terr := parseTime(s); terr == nil {
				track.MetadataTime = t
				track.HasMetadataTime = true
			}
		case "trkseg":
			// Observed, never decoded: handing this to DecodeElement would
			// swallow its <trkpt> children whole and they would never reach the
			// case below. Multiple <trk> elements need no separate handling —
			// each carries its own segments, so counting these is enough.
			pendingSegment = true
		case "trkpt":
			if len(track.Points) >= maxPoints {
				return Track{}, fmt.Errorf("gpx exceeds the maximum of %d track points", maxPoints)
			}
			var xp xmlTrkpt
			if err := dec.DecodeElement(&xp, &se); err != nil {
				return Track{}, fmt.Errorf("parsing track point: %w", err)
			}
			consumed = true
			pt, err := xp.toPoint()
			if err != nil {
				return Track{}, err
			}
			if pendingSegment && sawPoint {
				segIndex++
			}
			pendingSegment, sawPoint = false, true
			pt.SegmentIndex = segIndex
			track.Points = append(track.Points, pt)
		}

		if consumed {
			path = path[:len(path)-1]
		}
	}

	if !sawGPX {
		return Track{}, errors.New("not a GPX file: missing <gpx> root element")
	}
	track.finalize()
	return track, nil
}

// maxIdentityRunes bounds the descriptive strings copied out of a document.
// The upload cap is measured in megabytes, so a <trk><name> long enough to be a
// denial-of-service payload is well within a legal file; the parse boundary is
// where that gets stopped, as it already is for size and point count.
const maxIdentityRunes = 256

// parentIs reports whether the element currently being handled — the last entry
// in path — is a direct child of an element with the given local name.
func parentIs(path []string, parent string) bool {
	return len(path) >= 2 && path[len(path)-2] == parent
}

// decodeString reads an element's character data as a trimmed, bounded string.
// The error is deliberately free of the element's own content: a document may
// carry a hostile or entity-expanded body, and echoing it into an error message
// would carry it straight back out to the user.
func decodeString(dec *xml.Decoder, se *xml.StartElement) (string, error) {
	var s string
	if err := dec.DecodeElement(&s, se); err != nil {
		return "", fmt.Errorf("parsing <%s> element: %w", se.Name.Local, err)
	}
	return cleanIdentity(s), nil
}

// cleanIdentity trims a descriptive string and bounds its length. A
// whitespace-only value is treated as absent rather than as an empty name.
func cleanIdentity(s string) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > maxIdentityRunes {
		s = string([]rune(s)[:maxIdentityRunes])
	}
	return s
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
