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
	// FR-010: no elevation means no effort figure, and never a misleading 0.00.
	if !strings.Contains(out, "Descending elev.:  unavailable") {
		t.Errorf("expected descent unavailable notice:\n%s", out)
	}
	if !strings.Contains(out, "Effort km:         unavailable") {
		t.Errorf("expected effort unavailable notice:\n%s", out)
	}
	if strings.Contains(out, "0.00") {
		t.Errorf("must not render a zero effort figure when elevation is absent:\n%s", out)
	}
}

// effortResult is the SC-001 reference route: 10 km, 500 m of climb, 300 m of
// descent, so the two conventions read 15.00 and 16.00.
func effortResult() stats.Result {
	return stats.Result{
		TotalDistanceKm:      10,
		AscendingElevationM:  500,
		DescendingElevationM: 300,
		EffortKmClimb:        15,
		EffortKmClimbDescent: 16,
		HasElevation:         true,
		PointCount:           42,
	}
}

// TestWriteTextEffort pins the canonical labels, values and legend strings from
// contracts/ui-labels.md.
func TestWriteTextEffort(t *testing.T) {
	var buf bytes.Buffer
	cli.WriteText(&buf, effortResult())
	out := buf.String()

	for _, want := range []string{
		"Descending elev.:  300 m",
		"Effort km (climb):",
		"15.00",
		"100 m ascent = 1 km",
		"Effort km (climb + descent):",
		"16.00",
		"100 m ascent = 1 km, 300 m descent = 1 km",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

// TestWriteTextEffortWithoutTimes guards the !HasTimes early return: effort
// does not depend on timestamps, so a track with elevation but no times must
// still report both figures (FR-014).
func TestWriteTextEffortWithoutTimes(t *testing.T) {
	r := effortResult()
	r.HasTimes = false

	var buf bytes.Buffer
	cli.WriteText(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "Time-based stats:  unavailable") {
		t.Fatalf("fixture should have no time metrics:\n%s", out)
	}
	for _, want := range []string{
		"Descending elev.:  300 m",
		"Effort km (climb):",
		"Effort km (climb + descent):",
		"100 m ascent = 1 km, 300 m descent = 1 km",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("effort must be reported without timestamps, missing %q\n%s", want, out)
		}
	}
}
