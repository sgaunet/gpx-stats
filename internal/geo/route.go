// Package geo turns a parsed GPX track into display-ready map geometry.
//
// It is deliberately transport-agnostic: it imports only the standard library
// and internal/gpx, and knows nothing about HTTP, templates or JSON. The web
// layer adapts a Route to whatever wire format it needs (project constitution,
// Principle II).
//
// Two problems are solved here that would otherwise have to live, untested, in
// browser JavaScript:
//
//   - Antimeridian continuity. A track crossing 180 degrees longitude has raw
//     coordinates that jump from +179.x to -179.x. Drawn naively, the line
//     sweeps all the way across the world. Longitudes are therefore unwrapped
//     into a continuous sequence that may legitimately fall outside
//     [-180, 180]; map libraries accept this and wrap it for display.
//
//   - Degenerate extents. Framing a single point, or a track a few metres
//     across, by its bounding box asks for an impossible zoom level. Such
//     routes report UseBounds == false and are centred instead.
package geo

import (
	"math"

	"github.com/sgaunet/gpx-stats/internal/gpx"
)

const (
	// coordScale is 10^5, fixing coordinate output at five decimal places.
	// That is ~1.1 m at the equator: fine enough that consecutive GPS samples
	// never collapse together, and roughly half the payload size of full
	// float64 precision. It is the precision floor FR-001a permits.
	coordScale = 1e5

	// minExtentDegrees is the smallest bounding-box side worth framing with.
	// ~0.0005 degrees is ~55 m; below that, fitting the bounds would request a
	// zoom level beyond what tile providers supply.
	minExtentDegrees = 0.0005

	// DefaultPointZoom is the zoom level used when an extent is too small to
	// frame — a single point, or a track only metres across.
	DefaultPointZoom = 15

	// halfTurn is the longitude difference between consecutive points that
	// indicates the antimeridian was crossed rather than genuine movement.
	halfTurn = 180.0

	// fullTurn is the correction applied when unwrapping such a crossing.
	fullTurn = 360.0
)

// Route is a track prepared for display on a map.
//
// Every parsed track point is represented: no simplification, decimation or
// smoothing is applied at any size (spec 002-map-view, FR-001a). Callers that
// need a smaller payload must compress it, not reduce it.
type Route struct {
	// Segments holds one polyline per recorded track segment, in order. Each
	// segment is its own slice of [lat, lon] pairs, and every segment holds at
	// least one point — an empty <trkseg> never reaches here.
	//
	// Consecutive segments are deliberately NOT joined. A boundary means
	// recording was interrupted, so the straight line between the two was never
	// travelled and must not be drawn.
	//
	// Latitudes lie in [-90, 90]. Longitudes may fall outside [-180, 180]
	// after antimeridian unwrapping — this is intentional; do not "correct" it.
	Segments [][][2]float64

	// Bounds is the geographic extent as [minLat, minLon, maxLat, maxLon],
	// computed from the unwrapped longitudes. Meaningful only when UseBounds.
	Bounds [4]float64

	// UseBounds reports whether the extent is large enough to frame with.
	// When false, callers centre on Center at DefaultPointZoom instead.
	UseBounds bool

	// Center is the fallback view centre as [lat, lon].
	Center [2]float64

	// PointCount is the total number of points across every segment. Zero means
	// there is nothing to draw and the caller should omit the map entirely; one
	// means a lone marker and no line.
	PointCount int
}

// BuildRoute converts a parsed track into display-ready geometry.
//
// A track with no points yields the zero Route, which callers render as "no
// map" rather than as an empty one.
func BuildRoute(t gpx.Track) Route {
	if len(t.Points) == 0 {
		return Route{}
	}

	r := Route{PointCount: len(t.Points)}

	// Single pass: unwrap longitude, round, split on segment boundaries, and
	// accumulate the extent.
	var offset, prevRaw float64
	minLat, minLon := math.Inf(1), math.Inf(1)
	maxLat, maxLon := math.Inf(-1), math.Inf(-1)

	var cur [][2]float64
	for i, p := range t.Points {
		if i > 0 {
			// The unwrapping state is deliberately carried ACROSS segment
			// boundaries. The offset records where the track sits relative to
			// the antimeridian; resetting it per segment would place a dateline
			// track's second segment a full turn from its first and frame the
			// whole globe.
			switch d := p.Lon - prevRaw; {
			case d > halfTurn:
				// Jumped east-to-west across the antimeridian (+179 -> -179).
				offset -= fullTurn
			case d < -halfTurn:
				// Jumped west-to-east across it (-179 -> +179).
				offset += fullTurn
			}
			if !gpx.SameSegment(t.Points[i-1], p) {
				// cur always holds at least the previous point here, so no
				// empty segment can be emitted.
				r.Segments = append(r.Segments, cur)
				cur = nil
			}
		}
		prevRaw = p.Lon

		lat := round(p.Lat)
		lon := round(p.Lon + offset)
		cur = append(cur, [2]float64{lat, lon})

		minLat, maxLat = math.Min(minLat, lat), math.Max(maxLat, lat)
		minLon, maxLon = math.Min(minLon, lon), math.Max(maxLon, lon)
	}
	r.Segments = append(r.Segments, cur)

	r.Bounds = [4]float64{minLat, minLon, maxLat, maxLon}
	r.Center = [2]float64{(minLat + maxLat) / 2, (minLon + maxLon) / 2}
	r.UseBounds = maxLat-minLat >= minExtentDegrees || maxLon-minLon >= minExtentDegrees

	return r
}

// round returns v rounded (not truncated) to five decimal places.
func round(v float64) float64 {
	return math.Round(v*coordScale) / coordScale
}
