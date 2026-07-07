package stats_test

import (
	"math"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

func computeBase() time.Time { return time.Date(2023, 6, 15, 8, 0, 0, 0, time.UTC) }

// fullTrack mirrors testdata/sample.gpx: eight points with a 70s pause.
func fullTrack() gpx.Track {
	base := computeBase()
	mk := func(lon, ele float64, sec int) gpx.TrackPoint {
		return gpx.TrackPoint{
			Lat: 45, Lon: lon, Ele: ele, HasEle: true,
			HasTime: true, Time: base.Add(time.Duration(sec) * time.Second),
		}
	}
	return gpx.Track{
		HasElevation: true, HasTimes: true,
		Points: []gpx.TrackPoint{
			mk(6.0000, 100, 0),
			mk(6.0004, 105, 10),
			mk(6.0008, 110, 20),
			mk(6.0012, 112, 30),
			mk(6.0012, 112, 40),
			mk(6.0012, 112, 100),
			mk(6.0016, 118, 110),
			mk(6.0020, 125, 120),
		},
	}
}

func TestComputeFull(t *testing.T) {
	res := stats.Compute(fullTrack(), config.Default())

	if res.PointCount != 8 {
		t.Errorf("PointCount = %d, want 8", res.PointCount)
	}
	if res.TotalTime != 120*time.Second {
		t.Errorf("TotalTime = %s, want 120s", res.TotalTime)
	}
	if res.PauseCount != 1 {
		t.Errorf("PauseCount = %d, want 1", res.PauseCount)
	}
	if res.PauseTime != 70*time.Second {
		t.Errorf("PauseTime = %s, want 70s", res.PauseTime)
	}
	if res.MovingTime != 50*time.Second {
		t.Errorf("MovingTime = %s, want 50s", res.MovingTime)
	}
	// Core invariant.
	if res.MovingTime+res.PauseTime != res.TotalTime {
		t.Errorf("invariant broken: moving %s + pause %s != total %s",
			res.MovingTime, res.PauseTime, res.TotalTime)
	}
	if !res.HasElevation || res.AscendingElevationM <= 0 {
		t.Errorf("expected positive ascending elevation, got %g", res.AscendingElevationM)
	}
	if res.TotalDistanceKm <= 0 {
		t.Errorf("expected positive distance, got %g", res.TotalDistanceKm)
	}
	if res.AvgMovingSpeedKmh <= res.AvgSpeedKmh {
		t.Errorf("moving speed (%g) should exceed overall speed (%g)",
			res.AvgMovingSpeedKmh, res.AvgSpeedKmh)
	}
	if len(res.Splits) == 0 {
		t.Errorf("expected at least one split")
	}
}

func TestComputeNoTimes(t *testing.T) {
	track := gpx.Track{
		HasElevation: true, HasTimes: false,
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.0000, Ele: 100, HasEle: true},
			{Lat: 45, Lon: 6.0010, Ele: 110, HasEle: true},
		},
	}
	res := stats.Compute(track, config.Default())
	if res.HasTimes {
		t.Errorf("HasTimes should be false")
	}
	if res.TotalTime != 0 || res.MovingTime != 0 || res.PauseTime != 0 {
		t.Errorf("time metrics should be zero when timestamps are absent")
	}
	if len(res.Splits) != 0 {
		t.Errorf("no splits expected without timestamps")
	}
	if res.TotalDistanceKm <= 0 {
		t.Errorf("distance should still be computed, got %g", res.TotalDistanceKm)
	}
	if !res.HasElevation || res.AscendingElevationM <= 0 {
		t.Errorf("elevation should still be computed")
	}
}

func TestComputeEmpty(t *testing.T) {
	res := stats.Compute(gpx.Track{}, config.Default())
	if res.PointCount != 0 || res.TotalDistanceKm != 0 {
		t.Errorf("empty track should yield zero values, got %+v", res)
	}
}

func TestComputeSinglePoint(t *testing.T) {
	track := gpx.Track{
		Points: []gpx.TrackPoint{{Lat: 45, Lon: 6, Ele: 100, HasEle: true}},
	}
	res := stats.Compute(track, config.Default())
	if res.PointCount != 1 {
		t.Errorf("PointCount = %d, want 1", res.PointCount)
	}
	if res.TotalDistanceKm != 0 {
		t.Errorf("single point distance = %g, want 0", res.TotalDistanceKm)
	}
}

func TestComputeMovingPlusPauseAlwaysEqualsTotal(t *testing.T) {
	res := stats.Compute(fullTrack(), config.Default())
	if d := res.TotalTime - (res.MovingTime + res.PauseTime); math.Abs(float64(d)) > 0 {
		t.Errorf("moving + pause must equal total, off by %s", d)
	}
}
