package stats_test

import (
	"testing"

	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// TestDistanceAccuracy checks total distance against a known reference track
// (SC-003: within 1%). Paris → London: the WGS84 geodesic reference distance is
// ~343.9 km; the spherical haversine used here should land within 1%.
func TestDistanceAccuracy(t *testing.T) {
	const referenceKm = 343.9
	track := gpx.Track{
		Points: []gpx.TrackPoint{
			{Lat: 48.8566, Lon: 2.3522},  // Paris
			{Lat: 51.5074, Lon: -0.1278}, // London
		},
	}
	res := stats.Compute(track, config.Default())
	diffPct := (res.TotalDistanceKm - referenceKm) / referenceKm * 100
	if diffPct < 0 {
		diffPct = -diffPct
	}
	if diffPct > 1.0 {
		t.Errorf("distance %.2f km differs from reference %.1f km by %.2f%% (want <= 1%%)",
			res.TotalDistanceKm, referenceKm, diffPct)
	}
}
