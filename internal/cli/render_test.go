package cli_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

func TestWriteTextFull(t *testing.T) {
	r := stats.Result{
		TotalDistanceKm:     5.5,
		AscendingElevationM: 120,
		HasElevation:        true,
		TotalTime:           2 * time.Hour,
		MovingTime:          time.Hour + 50*time.Minute,
		PauseTime:           10 * time.Minute,
		PauseCount:          2,
		HasTimes:            true,
		AvgSpeedKmh:         2.75,
		AvgMovingSpeedKmh:   3.0,
		PointCount:          100,
		Splits: []stats.KmSplit{
			{Index: 1, DistanceKm: 1, Duration: 20 * time.Minute, SpeedKmh: 3},
		},
	}
	var buf bytes.Buffer
	cli.WriteText(&buf, r)
	out := buf.String()
	for _, want := range []string{"Total distance:", "5.50 km", "120 m", "Moving time:", "Kilometer splits:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestWriteTextUnavailable(t *testing.T) {
	r := stats.Result{TotalDistanceKm: 1.0, HasElevation: false, HasTimes: false, PointCount: 3}
	var buf bytes.Buffer
	cli.WriteText(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "Ascending elev.:   unavailable") {
		t.Errorf("expected elevation unavailable notice:\n%s", out)
	}
	if !strings.Contains(out, "Time-based stats:  unavailable") {
		t.Errorf("expected time unavailable notice:\n%s", out)
	}
	// Must not fabricate a zero moving time when times are absent.
	if strings.Contains(out, "Moving time:") {
		t.Errorf("should not print time metrics when unavailable:\n%s", out)
	}
}
