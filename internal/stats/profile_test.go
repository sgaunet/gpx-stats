package stats_test

import (
	"math"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// ramp builds a track whose points march east along the equator at a fixed
// longitude step, with the given elevations and (optionally) evenly-spaced
// timestamps one minute apart.
func ramp(elevations []float64, withTime bool) gpx.Track {
	base := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	pts := make([]gpx.TrackPoint, len(elevations))
	for i, e := range elevations {
		pts[i] = gpx.TrackPoint{
			Lat:    0,
			Lon:    float64(i) * 0.01,
			Ele:    e,
			HasEle: true,
		}
		if withTime {
			pts[i].Time = base.Add(time.Duration(i) * time.Minute)
			pts[i].HasTime = true
		}
	}
	return gpx.Track{Points: pts, HasElevation: true, HasTimes: withTime}
}

func TestElevationOverDistanceResamples(t *testing.T) {
	// Straight linear climb over evenly-spaced points: resampling onto 5 points
	// should reproduce a straight line, endpoints exact.
	track := ramp([]float64{100, 110, 120, 130, 140}, false)
	prof, ok := stats.ElevationOverDistance(track, 5)
	if !ok {
		t.Fatal("expected a distance profile")
	}
	if len(prof.Elevations) != 5 {
		t.Fatalf("want 5 samples, got %d", len(prof.Elevations))
	}
	if prof.Elevations[0] != 100 {
		t.Errorf("first sample = %v, want 100", prof.Elevations[0])
	}
	if prof.Elevations[len(prof.Elevations)-1] != 140 {
		t.Errorf("last sample = %v, want 140", prof.Elevations[len(prof.Elevations)-1])
	}
	// Points are equally spaced in distance, so the midpoint interpolates to 120.
	if math.Abs(prof.Elevations[2]-120) > 1e-6 {
		t.Errorf("mid sample = %v, want ~120", prof.Elevations[2])
	}
	if prof.XMax <= 0 {
		t.Errorf("XMax = %v, want > 0", prof.XMax)
	}
}

func TestElevationOverTimeResamples(t *testing.T) {
	track := ramp([]float64{100, 120, 140}, true)
	prof, ok := stats.ElevationOverTime(track, 3)
	if !ok {
		t.Fatal("expected a time profile")
	}
	// 3 points one minute apart -> 120 seconds of span.
	if math.Abs(prof.XMax-120) > 1e-6 {
		t.Errorf("XMax = %v seconds, want 120", prof.XMax)
	}
	if math.Abs(prof.Elevations[1]-120) > 1e-6 {
		t.Errorf("mid sample = %v, want ~120", prof.Elevations[1])
	}
}

func TestElevationOverTimeNeedsTimestamps(t *testing.T) {
	track := ramp([]float64{100, 110, 120}, false) // HasTimes=false
	if _, ok := stats.ElevationOverTime(track, 10); ok {
		t.Error("expected no time profile without timestamps")
	}
}

func TestElevationProfileUnavailable(t *testing.T) {
	t.Run("no elevation", func(t *testing.T) {
		track := gpx.Track{Points: []gpx.TrackPoint{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 0.01}}}
		if _, ok := stats.ElevationOverDistance(track, 10); ok {
			t.Error("expected no profile without elevation")
		}
	})
	t.Run("single point", func(t *testing.T) {
		track := gpx.Track{HasElevation: true, Points: []gpx.TrackPoint{{Ele: 100, HasEle: true}}}
		if _, ok := stats.ElevationOverDistance(track, 10); ok {
			t.Error("expected no profile from a single point")
		}
	})
	t.Run("zero distance span", func(t *testing.T) {
		// All points share the same coordinate: distance never advances.
		track := gpx.Track{HasElevation: true, Points: []gpx.TrackPoint{
			{Lat: 0, Lon: 0, Ele: 100, HasEle: true},
			{Lat: 0, Lon: 0, Ele: 120, HasEle: true},
		}}
		if _, ok := stats.ElevationOverDistance(track, 10); ok {
			t.Error("expected no profile when the track does not move")
		}
	})
}
