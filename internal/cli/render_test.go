package cli_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// hasRow reports whether out contains a row with this label, value and unit,
// whatever the column widths currently are.
//
// Assertions used to spell the padding out by hand, which meant adding one label
// rewrote the expectations of twenty unrelated lines. This asserts what the
// layout actually promises — one line, label then value then optional unit — and
// the trailing anchor is what proves nothing else was appended to the row.
func hasRow(out, label, value, unit string) bool {
	pat := `(?m)^ *` + regexp.QuoteMeta(label) + ` {2,}` + regexp.QuoteMeta(value)
	if unit != "" {
		pat += ` {2,}` + regexp.QuoteMeta(unit)
	}
	return regexp.MustCompile(pat + `$`).MatchString(out)
}

// hasLabel reports whether out carries a row for this label at all, which is
// what the "must not appear" assertions actually mean. A bare
// strings.Contains(out, "Type") would match any line merely mentioning the word.
func hasLabel(out, label string) bool {
	return regexp.MustCompile(`(?m)^ *` + regexp.QuoteMeta(label) + ` {2,}`).MatchString(out)
}

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
	for _, want := range []struct{ label, value, unit string }{
		{"Total distance", "5.50", "km"},
		{"Ascending elev.", "120", "m"},
		{"Moving time", "1h50m00s", ""},
	} {
		if !hasRow(out, want.label, want.value, want.unit) {
			t.Errorf("output missing row %q %q %q\n%s", want.label, want.value, want.unit, out)
		}
	}
	if !strings.Contains(out, "Kilometer splits") {
		t.Errorf("output missing the splits section\n%s", out)
	}
}

// noElevation is the canonical unavailable wording from contracts/ui-labels.md.
const noElevation = "unavailable (no elevation data)"

func TestWriteTextUnavailable(t *testing.T) {
	r := stats.Result{TotalDistanceKm: 1.0, HasElevation: false, HasTimes: false, PointCount: 3}
	var buf bytes.Buffer
	cli.WriteText(&buf, r)
	out := buf.String()
	if !hasRow(out, "Ascending elev.", noElevation, "") {
		t.Errorf("expected elevation unavailable notice:\n%s", out)
	}
	if !hasRow(out, "Time-based stats", "unavailable (no timestamps)", "") {
		t.Errorf("expected time unavailable notice:\n%s", out)
	}
	// Must not fabricate a zero moving time when times are absent.
	if hasLabel(out, "Moving time") {
		t.Errorf("should not print time metrics when unavailable:\n%s", out)
	}
	// FR-010: no elevation means no effort figure, and never a misleading 0.00.
	if !hasRow(out, "Descending elev.", noElevation, "") {
		t.Errorf("expected descent unavailable notice:\n%s", out)
	}
	if !hasRow(out, "Effort km", noElevation, "") {
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

	for _, want := range []struct{ label, value, unit string }{
		{"Descending elev.", "300", "m"},
		{"Effort km (climb)", "15.00", ""},
		{"Effort km (climb + descent)", "16.00", ""},
	} {
		if !hasRow(out, want.label, want.value, want.unit) {
			t.Errorf("output missing row %q %q %q\n%s", want.label, want.value, want.unit, out)
		}
	}
	for _, want := range []string{
		"100 m ascent = 1 km",
		"100 m ascent = 1 km, 300 m descent = 1 km",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing legend %q\n%s", want, out)
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

	if !hasRow(out, "Time-based stats", "unavailable (no timestamps)", "") {
		t.Fatalf("fixture should have no time metrics:\n%s", out)
	}
	for _, want := range []struct{ label, value, unit string }{
		{"Descending elev.", "300", "m"},
		{"Effort km (climb)", "15.00", ""},
		{"Effort km (climb + descent)", "16.00", ""},
	} {
		if !hasRow(out, want.label, want.value, want.unit) {
			t.Errorf("effort must be reported without timestamps, missing %q\n%s", want.label, out)
		}
	}
	if !strings.Contains(out, "100 m ascent = 1 km, 300 m descent = 1 km") {
		t.Errorf("effort legend must survive without timestamps\n%s", out)
	}
	// The rates do depend on timestamps, so they must be absent rather than
	// rendered as a zero the reader would take for a measurement.
	for _, unwanted := range []string{"Effort km/h", "Moving effort km/h"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("effort rate %q must not appear without timestamps\n%s", unwanted, out)
		}
	}
}

// effortRateResult adds timestamps to the reference route so all four rates are
// available. The values are distinct from the totals and from each other, so a
// label wired to the wrong field fails rather than coincidentally matching.
func effortRateResult() stats.Result {
	r := effortResult()
	r.HasTimes = true
	r.TotalTime = time.Hour
	r.MovingTime = 45 * time.Minute
	r.EffortSpeedKmhClimb = 12.5
	r.EffortMovingSpeedKmhClimb = 13.2
	r.EffortSpeedKmhClimbDescent = 13.33
	r.EffortMovingSpeedKmhClimbDescent = 14.1
	return r
}

// TestWriteTextEffortRates pins the canonical rate labels from
// contracts/ui-labels.md and their placement: each convention's rates sit
// between its total and its legend, so one legend covers all three figures.
func TestWriteTextEffortRates(t *testing.T) {
	var buf bytes.Buffer
	cli.WriteText(&buf, effortRateResult())
	out := buf.String()

	for _, want := range []struct{ label, value string }{
		{"Effort km/h (climb)", "12.50"},
		{"Moving effort km/h (climb)", "13.20"},
		{"Effort km/h (climb + descent)", "13.33"},
		{"Moving effort km/h (climb + descent)", "14.10"},
	} {
		if !hasRow(out, want.label, want.value, "km/h") {
			t.Errorf("output missing rate row %q %q\n%s", want.label, want.value, out)
		}
	}

	// Ordering: total, its two rates, then the legend that explains all three.
	// The "\n" on the climb legend is what stops it matching the longer
	// climb+descent legend, which starts with the same text. It holds only
	// because the test writer is a buffer and therefore unstyled; with styling
	// on, a reset would sit between the text and the newline.
	order := []string{
		"Effort km (climb)",
		"Effort km/h (climb)",
		"Moving effort km/h (climb)",
		"100 m ascent = 1 km\n",
		"Effort km (climb + descent)",
		"Effort km/h (climb + descent)",
		"Moving effort km/h (climb + descent)",
		"100 m ascent = 1 km, 300 m descent = 1 km",
	}
	prev := -1
	for _, s := range order {
		at := strings.Index(out, s)
		if at <= prev {
			t.Fatalf("%q is out of order (at %d, previous at %d)\n%s", s, at, prev, out)
		}
		prev = at
	}

	// Each legend is stated exactly once, however many figures it covers.
	if n := strings.Count(out, "100 m ascent = 1 km, 300 m descent = 1 km"); n != 1 {
		t.Errorf("climb+descent legend appears %d times, want 1\n%s", n, out)
	}
}
