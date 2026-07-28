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

// closeTo reports whether a and b agree to within eps.
func closeTo(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestBuildRouteEmptyTrack(t *testing.T) {
	r := geo.BuildRoute(gpx.Track{})
	if r.PointCount != 0 {
		t.Errorf("PointCount = %d, want 0", r.PointCount)
	}
	if len(r.Coords) != 0 {
		t.Errorf("Coords = %v, want empty", r.Coords)
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
	if len(r.Coords) != 2 {
		t.Errorf("len(Coords) = %d, want 2", len(r.Coords))
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
	for i := 0; i < len(r.Coords); i += 2 {
		lat, lon := r.Coords[i], r.Coords[i+1]
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
	for i := 3; i < len(r.Coords); i += 2 {
		if r.Coords[i] <= r.Coords[i-2] {
			t.Errorf("longitude sequence not monotonic after unwrapping: %v", r.Coords)
			break
		}
	}
	if last := r.Coords[len(r.Coords)-1]; !closeTo(last, 180.10, 1e-6) {
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
	if last := r.Coords[len(r.Coords)-1]; !closeTo(last, -180.10, 1e-6) {
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
	for i := 3; i < len(r.Coords); i += 2 {
		if d := math.Abs(r.Coords[i] - r.Coords[i-2]); d > 180 {
			t.Errorf("consecutive longitudes differ by %v (>180): %v", d, r.Coords)
			break
		}
	}
}

func TestBuildRouteRoundsToFiveDecimals(t *testing.T) {
	r := geo.BuildRoute(track(45.123456789, 6.987654321))
	// 5 dp is ~1.1 m, the floor FR-001a permits. Values are rounded, not truncated.
	if got, want := r.Coords[0], 45.12346; !closeTo(got, want, 1e-9) {
		t.Errorf("lat = %v, want %v", got, want)
	}
	if got, want := r.Coords[1], 6.98765; !closeTo(got, want, 1e-9) {
		t.Errorf("lon = %v, want %v", got, want)
	}
}

func TestBuildRoutePrecisionFloorKeepsPointsDistinct(t *testing.T) {
	// ~1.2 m apart in latitude (1 degree ~ 111320 m): must survive rounding,
	// proving 5 decimal places is enough precision for real GPS traces.
	const delta = 1.2 / 111320.0
	r := geo.BuildRoute(track(45.0, 6.0, 45.0+delta, 6.0))
	if r.Coords[0] == r.Coords[2] {
		t.Errorf("two points 1.2 m apart collapsed to the same latitude %v", r.Coords[0])
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
		if len(r.Coords) != n*2 {
			t.Errorf("len(Coords) = %d, want %d", len(r.Coords), n*2)
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

func TestBuildRouteCoordsAlwaysEvenLength(t *testing.T) {
	for _, n := range []int{0, 1, 2, 5} {
		latlons := make([]float64, 0, n*2)
		for i := range n {
			latlons = append(latlons, 45.0+float64(i), 6.0+float64(i))
		}
		r := geo.BuildRoute(track(latlons...))
		if len(r.Coords)%2 != 0 {
			t.Errorf("n=%d: len(Coords) = %d, want an even length", n, len(r.Coords))
		}
	}
}
