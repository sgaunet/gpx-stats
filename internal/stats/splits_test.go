package stats_test

import (
	"math"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

func TestKilometerSplits(t *testing.T) {
	base := time.Date(2023, 6, 15, 8, 0, 0, 0, time.UTC)
	// Points along the equator ~0.6 km apart, 360 s each → a steady 6 km/h.
	deg := 0.6 / 111.195 // degrees of longitude per 0.6 km at the equator
	pts := []gpx.TrackPoint{
		{Lat: 0, Lon: 0, HasTime: true, Time: base},
		{Lat: 0, Lon: deg, HasTime: true, Time: base.Add(360 * time.Second)},
		{Lat: 0, Lon: 2 * deg, HasTime: true, Time: base.Add(720 * time.Second)},
	}
	splits := stats.KilometerSplits(pts)
	if len(splits) != 2 {
		t.Fatalf("got %d splits, want 2 (one full km + partial)", len(splits))
	}
	if splits[0].Index != 1 || splits[1].Index != 2 {
		t.Errorf("split indexes = %d,%d want 1,2", splits[0].Index, splits[1].Index)
	}
	if math.Abs(splits[0].DistanceKm-1.0) > 0.02 {
		t.Errorf("first split distance = %g, want ~1.0", splits[0].DistanceKm)
	}
	if splits[1].DistanceKm >= 1.0 {
		t.Errorf("final split should be partial, got %g km", splits[1].DistanceKm)
	}
	for _, s := range splits {
		if math.Abs(s.SpeedKmh-6.0) > 0.2 {
			t.Errorf("split %d speed = %g, want ~6.0", s.Index, s.SpeedKmh)
		}
	}
	totalDist := splits[0].DistanceKm + splits[1].DistanceKm
	if math.Abs(totalDist-1.2) > 0.02 {
		t.Errorf("sum of split distances = %g, want ~1.2", totalDist)
	}
}

func TestKilometerSplitsIncludesStationaryTime(t *testing.T) {
	base := time.Date(2023, 6, 15, 8, 0, 0, 0, time.UTC)
	deg := 0.3 / 111.195
	// Move 0.3 km (60s), stop 100s in place, move 0.3 km (60s): all < 1 km,
	// so one partial split whose duration includes the stop.
	pts := []gpx.TrackPoint{
		{Lat: 0, Lon: 0, HasTime: true, Time: base},
		{Lat: 0, Lon: deg, HasTime: true, Time: base.Add(60 * time.Second)},
		{Lat: 0, Lon: deg, HasTime: true, Time: base.Add(160 * time.Second)},
		{Lat: 0, Lon: 2 * deg, HasTime: true, Time: base.Add(220 * time.Second)},
	}
	splits := stats.KilometerSplits(pts)
	if len(splits) != 1 {
		t.Fatalf("got %d splits, want 1", len(splits))
	}
	if splits[0].Duration != 220*time.Second {
		t.Errorf("split duration = %s, want 220s (includes stop)", splits[0].Duration)
	}
}
