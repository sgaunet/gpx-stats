package geo_test

import (
	"math"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/geo"
	"github.com/sgaunet/gpx-stats/internal/gpx"
)

// track builds a gpx.Track from flat lat/lon pairs.
func track(latlons ...float64) gpx.Track {
	if len(latlons)%2 != 0 {
		panic("track: odd number of coordinates")
	}
	var t gpx.Track
	for i := 0; i < len(latlons); i += 2 {
		t.Points = append(t.Points, gpx.TrackPoint{Lat: latlons[i], Lon: latlons[i+1]})
	}
	return t
}

// segTrack builds a gpx.Track from (segment, lat, lon) triples.
func segTrack(triples ...float64) gpx.Track {
	if len(triples)%3 != 0 {
		panic("segTrack: coordinates must come in (segment, lat, lon) triples")
	}
	var t gpx.Track
	for i := 0; i < len(triples); i += 3 {
		t.Points = append(t.Points, gpx.TrackPoint{
			SegmentIndex: int(triples[i]), Lat: triples[i+1], Lon: triples[i+2],
		})
	}
	return t
}

// flat re-flattens a route into the [lat0, lon0, lat1, lon1, ...] form the
// geometry assertions below were written against. Unwrapping, rounding and
// framing are independent of how the points are grouped; the grouping itself is
// asserted separately by the segment tests.
func flat(r geo.Route) []float64 {
	var out []float64
	for _, seg := range r.Segments {
		for _, p := range seg {
			out = append(out, p[0], p[1])
		}
	}
	return out
}

// closeTo reports whether a and b agree to within eps.
func closeTo(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestBuildRouteEmptyTrack(t *testing.T) {
	r := geo.BuildRoute(gpx.Track{})
	if r.PointCount != 0 {
		t.Errorf("PointCount = %d, want 0", r.PointCount)
	}
	if len(r.Segments) != 0 {
		t.Errorf("Segments = %v, want none for an empty track", r.Segments)
	}
	if r.UseBounds {
		t.Errorf("UseBounds = true, want false for an empty track")
	}
}

func TestBuildRouteSinglePoint(t *testing.T) {
	r := geo.BuildRoute(track(45.5, 6.25))
	if r.PointCount != 1 {
		t.Fatalf("PointCount = %d, want 1", r.PointCount)
	}
	if r.UseBounds {
		t.Errorf("UseBounds = true, want false: fitBounds on one point would over-zoom")
	}
	if !closeTo(r.Center[0], 45.5, 1e-9) || !closeTo(r.Center[1], 6.25, 1e-9) {
		t.Errorf("Center = %v, want [45.5 6.25]", r.Center)
	}
	if got := flat(r); len(got) != 2 {
		t.Errorf("flattened length = %d, want 2", len(got))
	}
}

func TestBuildRouteCoincidentPoints(t *testing.T) {
	r := geo.BuildRoute(track(45.5, 6.25, 45.5, 6.25))
	if r.PointCount != 2 {
		t.Fatalf("PointCount = %d, want 2", r.PointCount)
	}
	if r.UseBounds {
		t.Errorf("UseBounds = true, want false: a zero-size extent cannot be framed")
	}
	if !closeTo(r.Center[0], 45.5, 1e-9) || !closeTo(r.Center[1], 6.25, 1e-9) {
		t.Errorf("Center = %v, want the coincident point", r.Center)
	}
}

func TestBuildRouteTinyExtent(t *testing.T) {
	// ~10 m apart: below the minimum extent worth framing.
	r := geo.BuildRoute(track(45.0, 6.0, 45.00009, 6.0))
	if r.UseBounds {
		t.Errorf("UseBounds = true, want false for a ~10 m extent")
	}
	if r.PointCount != 2 {
		t.Errorf("PointCount = %d, want 2", r.PointCount)
	}
}

func TestBuildRouteNormalTrack(t *testing.T) {
	r := geo.BuildRoute(track(
		45.0, 6.0,
		45.1, 6.2,
		45.05, 6.3,
		44.9, 6.1,
	))
	if r.PointCount != 4 {
		t.Fatalf("PointCount = %d, want 4", r.PointCount)
	}
	if !r.UseBounds {
		t.Fatalf("UseBounds = false, want true for a spread track")
	}
	minLat, minLon, maxLat, maxLon := r.Bounds[0], r.Bounds[1], r.Bounds[2], r.Bounds[3]
	if !closeTo(minLat, 44.9, 1e-9) || !closeTo(maxLat, 45.1, 1e-9) {
		t.Errorf("latitude bounds = [%v %v], want [44.9 45.1]", minLat, maxLat)
	}
	if !closeTo(minLon, 6.0, 1e-9) || !closeTo(maxLon, 6.3, 1e-9) {
		t.Errorf("longitude bounds = [%v %v], want [6.0 6.3]", minLon, maxLon)
	}
	// Every point must fall inside the reported extent.
	c := flat(r)
	for i := 0; i < len(c); i += 2 {
		lat, lon := c[i], c[i+1]
		if lat < minLat || lat > maxLat || lon < minLon || lon > maxLon {
			t.Errorf("point [%v %v] falls outside bounds %v", lat, lon, r.Bounds)
		}
	}
}

func TestBuildRouteAntimeridianEastward(t *testing.T) {
	// Crossing 180 going east: raw longitudes jump +179.95 -> -179.95.
	r := geo.BuildRoute(track(
		-16.5, 179.90,
		-16.5, 179.95,
		-16.5, -179.95,
		-16.5, -179.90,
	))
	span := r.Bounds[3] - r.Bounds[1]
	if span > 1 {
		t.Errorf("longitude span = %v, want ~0.2: the track was not unwrapped and "+
			"would draw a line sweeping across the world", span)
	}
	// The unwrapped sequence must keep increasing past 180.
	c := flat(r)
	for i := 3; i < len(c); i += 2 {
		if c[i] <= c[i-2] {
			t.Errorf("longitude sequence not monotonic after unwrapping: %v", c)
			break
		}
	}
	if last := c[len(c)-1]; !closeTo(last, 180.10, 1e-6) {
		t.Errorf("final longitude = %v, want 180.10 (unwrapped)", last)
	}
}

func TestBuildRouteAntimeridianWestward(t *testing.T) {
	// Crossing 180 going west: raw longitudes jump -179.95 -> +179.95.
	r := geo.BuildRoute(track(
		-16.5, -179.90,
		-16.5, -179.95,
		-16.5, 179.95,
		-16.5, 179.90,
	))
	span := r.Bounds[3] - r.Bounds[1]
	if span > 1 {
		t.Errorf("longitude span = %v, want ~0.2", span)
	}
	c := flat(r)
	if last := c[len(c)-1]; !closeTo(last, -180.10, 1e-6) {
		t.Errorf("final longitude = %v, want -180.10 (unwrapped)", last)
	}
}

func TestBuildRouteAntimeridianMultipleCrossings(t *testing.T) {
	r := geo.BuildRoute(track(
		0, 179.9,
		0, -179.9, // east across
		0, 179.9, // back west
		0, -179.9, // east again
	))
	// No consecutive pair may differ by more than 180 degrees after unwrapping.
	c := flat(r)
	for i := 3; i < len(c); i += 2 {
		if d := math.Abs(c[i] - c[i-2]); d > 180 {
			t.Errorf("consecutive longitudes differ by %v (>180): %v", d, c)
			break
		}
	}
}

func TestBuildRouteRoundsToFiveDecimals(t *testing.T) {
	r := geo.BuildRoute(track(45.123456789, 6.987654321))
	// 5 dp is ~1.1 m, the floor FR-001a permits. Values are rounded, not truncated.
	c := flat(r)
	if got, want := c[0], 45.12346; !closeTo(got, want, 1e-9) {
		t.Errorf("lat = %v, want %v", got, want)
	}
	if got, want := c[1], 6.98765; !closeTo(got, want, 1e-9) {
		t.Errorf("lon = %v, want %v", got, want)
	}
}

func TestBuildRoutePrecisionFloorKeepsPointsDistinct(t *testing.T) {
	// ~1.2 m apart in latitude (1 degree ~ 111320 m): must survive rounding,
	// proving 5 decimal places is enough precision for real GPS traces.
	const delta = 1.2 / 111320.0
	r := geo.BuildRoute(track(45.0, 6.0, 45.0+delta, 6.0))
	c := flat(r)
	if c[0] == c[2] {
		t.Errorf("two points 1.2 m apart collapsed to the same latitude %v", c[0])
	}
}

func TestBuildRouteKeepsEveryPoint(t *testing.T) {
	// FR-001a: no simplification, decimation or stride-sampling at any size.
	for _, n := range []int{2, 1000, 100000} {
		latlons := make([]float64, 0, n*2)
		for i := range n {
			latlons = append(latlons, 45.0+float64(i)*1e-4, 6.0+float64(i)*1e-4)
		}
		r := geo.BuildRoute(track(latlons...))
		if r.PointCount != n {
			t.Errorf("PointCount = %d, want %d (points were dropped)", r.PointCount, n)
		}
		if got := flat(r); len(got) != n*2 {
			t.Errorf("flattened length = %d, want %d", len(got), n*2)
		}
	}
}

func TestBuildRouteHighLatitude(t *testing.T) {
	r := geo.BuildRoute(track(84.9, 10.0, 85.1, 10.5))
	if !r.UseBounds {
		t.Fatalf("UseBounds = false, want true")
	}
	// Latitude must never be wrapped the way longitude is.
	if !closeTo(r.Bounds[0], 84.9, 1e-9) || !closeTo(r.Bounds[2], 85.1, 1e-9) {
		t.Errorf("latitude bounds = [%v %v], want [84.9 85.1]", r.Bounds[0], r.Bounds[2])
	}
}

func TestBuildRouteSegmentsAccountForEveryPoint(t *testing.T) {
	for _, n := range []int{0, 1, 2, 5} {
		latlons := make([]float64, 0, n*2)
		for i := range n {
			latlons = append(latlons, 45.0+float64(i), 6.0+float64(i))
		}
		r := geo.BuildRoute(track(latlons...))
		total := 0
		for _, seg := range r.Segments {
			// map.js indexes seg[0] without guarding, so an empty segment would
			// be a client-side crash rather than a cosmetic problem.
			if len(seg) == 0 {
				t.Errorf("n=%d: an empty segment was emitted", n)
			}
			total += len(seg)
		}
		if total != r.PointCount {
			t.Errorf("n=%d: segments hold %d points, want PointCount %d", n, total, r.PointCount)
		}
	}
}

func TestBuildRouteSplitsOnSegmentBoundary(t *testing.T) {
	r := geo.BuildRoute(segTrack(
		0, 45.0, 6.0,
		0, 45.1, 6.1,
		1, 45.5, 6.5,
		1, 45.6, 6.6,
		1, 45.7, 6.7,
	))
	if len(r.Segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(r.Segments))
	}
	if len(r.Segments[0]) != 2 || len(r.Segments[1]) != 3 {
		t.Errorf("segment sizes = %d/%d, want 2/3", len(r.Segments[0]), len(r.Segments[1]))
	}
	if r.PointCount != 5 {
		t.Errorf("PointCount = %d, want 5 (framing still sees the whole track)", r.PointCount)
	}
	// Bounds span every segment, so the map still frames the whole activity.
	if !closeTo(r.Bounds[0], 45.0, 1e-9) || !closeTo(r.Bounds[2], 45.7, 1e-9) {
		t.Errorf("latitude bounds = [%v %v], want [45.0 45.7]", r.Bounds[0], r.Bounds[2])
	}
}

func TestBuildRouteEmptySegmentNeverEmitted(t *testing.T) {
	// Indices produced by the parser are dense, but a hand-built track may skip
	// one. Neither may yield an empty sub-array.
	r := geo.BuildRoute(segTrack(
		0, 45.0, 6.0,
		2, 45.5, 6.5,
	))
	if len(r.Segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(r.Segments))
	}
	for i, seg := range r.Segments {
		if len(seg) == 0 {
			t.Errorf("segment %d is empty", i)
		}
	}
}

func TestBuildRouteSingleSegmentIsOnePolyline(t *testing.T) {
	// The overwhelmingly common case must not become a list of one-point lines.
	r := geo.BuildRoute(track(45.0, 6.0, 45.1, 6.1, 45.2, 6.2))
	if len(r.Segments) != 1 {
		t.Fatalf("got %d segments, want 1", len(r.Segments))
	}
	if len(r.Segments[0]) != 3 {
		t.Errorf("segment holds %d points, want 3", len(r.Segments[0]))
	}
}

func TestBuildRouteAntimeridianAcrossSegmentBoundary(t *testing.T) {
	// A dateline track whose recording was interrupted mid-crossing. The
	// unwrapping offset must survive the boundary: resetting it would place the
	// second segment a full turn away and frame the entire globe.
	r := geo.BuildRoute(segTrack(
		0, -16.5, 179.90,
		0, -16.5, 179.95,
		1, -16.5, -179.95,
		1, -16.5, -179.90,
	))
	if len(r.Segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(r.Segments))
	}
	if span := r.Bounds[3] - r.Bounds[1]; span > 1 {
		t.Errorf("longitude span = %v, want ~0.2: the offset was reset at the segment "+
			"boundary and the track now spans the world", span)
	}
	if last := r.Segments[1][1][1]; !closeTo(last, 180.10, 1e-6) {
		t.Errorf("final longitude = %v, want 180.10 (unwrapped)", last)
	}
}
