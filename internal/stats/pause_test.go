package stats_test

import (
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

var pauseBase = time.Date(2023, 6, 15, 8, 0, 0, 0, time.UTC)

// tpt builds a timestamped point at latitude 45 and the given longitude.
func tpt(lon float64, sec int) gpx.TrackPoint {
	return gpx.TrackPoint{
		Lat:     45,
		Lon:     lon,
		HasTime: true,
		Time:    pauseBase.Add(time.Duration(sec) * time.Second),
	}
}

func TestDetectPausesSingleRun(t *testing.T) {
	// Move, then hold the same position from 10s to 80s (70s), then move.
	pts := []gpx.TrackPoint{
		tpt(6.0000, 0),
		tpt(6.0010, 10),
		tpt(6.0010, 20),
		tpt(6.0010, 80),
		tpt(6.0020, 90),
	}
	pauses, total := stats.DetectPauses(pts, 1.0, 10*time.Second)
	if len(pauses) != 1 {
		t.Fatalf("got %d pauses, want 1", len(pauses))
	}
	if total != 70*time.Second {
		t.Errorf("total pause = %s, want 70s", total)
	}
	if pauses[0].Duration != 70*time.Second {
		t.Errorf("pause duration = %s, want 70s", pauses[0].Duration)
	}
}

func TestDetectPausesBelowMinDurationIgnored(t *testing.T) {
	// Stationary for only 5s, below the 10s minimum.
	pts := []gpx.TrackPoint{
		tpt(6.0000, 0),
		tpt(6.0000, 5),
		tpt(6.0010, 15),
	}
	pauses, total := stats.DetectPauses(pts, 1.0, 10*time.Second)
	if len(pauses) != 0 {
		t.Fatalf("got %d pauses, want 0", len(pauses))
	}
	if total != 0 {
		t.Errorf("total pause = %s, want 0", total)
	}
}

func TestDetectPausesNoneWhenMoving(t *testing.T) {
	pts := []gpx.TrackPoint{
		tpt(6.0000, 0),
		tpt(6.0010, 10),
		tpt(6.0020, 20),
	}
	pauses, total := stats.DetectPauses(pts, 1.0, 10*time.Second)
	if len(pauses) != 0 || total != 0 {
		t.Errorf("expected no pauses, got %d (%s)", len(pauses), total)
	}
}

func TestDetectPausesOutOfOrderIgnored(t *testing.T) {
	// A backwards timestamp segment must be skipped without panicking, and the
	// remaining moving segments yield no pause.
	pts := []gpx.TrackPoint{
		tpt(6.0000, 0),
		tpt(6.0010, 10),
		tpt(6.0010, 5), // backwards
		tpt(6.0020, 30),
	}
	pauses, total := stats.DetectPauses(pts, 1.0, 10*time.Second)
	if len(pauses) != 0 {
		t.Errorf("expected no pauses for out-of-order input, got %d", len(pauses))
	}
	if total != 0 {
		t.Errorf("expected zero pause time, got %s", total)
	}
}
