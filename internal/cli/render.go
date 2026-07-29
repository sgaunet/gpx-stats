// Package cli implements the command-line transport for gpx-stats: flag
// parsing, dispatch, and rendering of statistics as text or JSON. It depends on
// the domain packages (gpx, stats, config) but they never depend on it.
package cli

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/sgaunet/gpx-stats/internal/stats"
)

// humanTimeLayout is the terminal's time format. The machine surfaces keep
// RFC3339 (see WriteJSON and the web templates); a person reading a terminal is
// better served by a space than by a "T".
const humanTimeLayout = "2006-01-02 15:04:05 MST"

const (
	noElevationNote = "unavailable (no elevation data)"
	noTimesNote     = "unavailable (no timestamps)"
	noIdentityNote  = "unavailable (no identity metadata in the file)"
)

// WriteText renders a Result as human-readable text to w. Unavailable metrics
// are labelled explicitly rather than shown as a misleading zero.
//
// Every row goes through a single table, which measures its own columns: the
// report has one label column and one right-aligned value column no matter which
// sections a given file happens to fill.
func WriteText(w io.Writer, r stats.Result, opts ...Option) {
	st := styleFor(w, opts)
	writeTitle(w, st, "GPX Statistics")

	t := &table{st: st}
	writeActivity(t, r.Activity)
	writeDistance(t, r)
	// Effort kilometers come before the time-based section on purpose: they are
	// derived from distance and elevation only, so a track without timestamps
	// must still report them.
	writeEffort(t, r)
	writeTime(t, r)
	t.flush(w)

	// Guarded by HasTimes, not by len(Splits) alone: a track without timestamps
	// has no meaningful split times. This guard used to be provided by an early
	// return that the table rewrite removed.
	if r.HasTimes && len(r.Splits) > 0 {
		writeSplits(w, st, r)
	}
}

// writeActivity renders whatever the file said about itself, one row per field
// that is actually present.
//
// A file carrying none of it gets a single "unavailable" row rather than six:
// the terminal is the human surface, and six identical negatives would push the
// statistics down the screen to say nothing. The machine-readable output is
// where every field is always present.
func writeActivity(t *table, a stats.Activity) {
	t.section("Activity")
	if a.Name == "" && a.Type == "" && a.Creator == "" && !a.HasMetadataTime && !a.HasStartEnd {
		t.text("Activity", noIdentityNote)
		return
	}
	if a.Name != "" {
		t.text("Activity", a.Name)
	}
	if a.Type != "" {
		t.text("Type", a.Type)
	}
	if a.Creator != "" {
		t.text("Recorded by", a.Creator)
	}
	if a.HasStartEnd {
		t.text("Start", a.Start.Format(humanTimeLayout))
		t.text("End", a.End.Format(humanTimeLayout))
	}
	if a.HasMetadataTime {
		// Labelled "File time" on purpose: it is when the document was written,
		// which exporters routinely set to the moment you downloaded it. Calling
		// it a date would misdate the activity.
		t.text("File time", a.MetadataTime.Format(humanTimeLayout))
	}
}

func writeDistance(t *table, r stats.Result) {
	t.section("Distance & elevation")
	t.num("Points", strconv.Itoa(r.PointCount), "")
	t.num("Total distance", f2(r.TotalDistanceKm), "km")
	// The segment note only appears when it explains something. A reader
	// comparing this figure against a tool that joins the segments would
	// otherwise have no way to account for the difference. It is a note rather
	// than a parenthetical because appended to the row it would wrap.
	if r.SegmentCount > 1 {
		t.note(fmt.Sprintf("%d segments; gaps between them are not counted", r.SegmentCount))
	}

	if !r.HasElevation {
		t.text("Ascending elev.", noElevationNote)
		t.text("Descending elev.", noElevationNote)
		return
	}
	t.num("Ascending elev.", fmt.Sprintf("%.0f", r.AscendingElevationM), "m")
	t.num("Descending elev.", fmt.Sprintf("%.0f", r.DescendingElevationM), "m")
}

// writeEffort renders the two effort-kilometer conventions, each with its
// per-hour rates and then its legend. Grouping by convention rather than by
// figure is what lets one legend cover a convention's total and both of its
// rates, instead of repeating the equivalence four times.
//
// Neither convention is presented as the correct one.
func writeEffort(t *table, r stats.Result) {
	t.section("Effort kilometers")
	if !r.HasElevation {
		t.text("Effort km", noElevationNote)
		return
	}
	t.num("Effort km (climb)", f2(r.EffortKmClimb), "")
	// The rates need timestamps; the totals above do not. A track without them
	// keeps its effort kilometers and is accounted for by the "Time-based stats:
	// unavailable" row that follows this section.
	if r.HasTimes {
		t.num("Effort km/h (climb)", f2(r.EffortSpeedKmhClimb), "km/h")
		t.num("Moving effort km/h (climb)", f2(r.EffortMovingSpeedKmhClimb), "km/h")
	}
	t.note("100 m ascent = 1 km")
	t.num("Effort km (climb + descent)", f2(r.EffortKmClimbDescent), "")
	if r.HasTimes {
		t.num("Effort km/h (climb + descent)", f2(r.EffortSpeedKmhClimbDescent), "km/h")
		t.num("Moving effort km/h (climb + descent)", f2(r.EffortMovingSpeedKmhClimbDescent), "km/h")
	}
	t.note("100 m ascent = 1 km, 300 m descent = 1 km")
}

func writeTime(t *table, r stats.Result) {
	t.section("Time & speed")
	if !r.HasTimes {
		t.text("Time-based stats", noTimesNote)
		return
	}
	t.num("Total time", formatDuration(r.TotalTime), "")
	t.num("Moving time", formatDuration(r.MovingTime), "")
	t.num("Pause time", formatDuration(r.PauseTime), fmt.Sprintf("(%d pauses)", r.PauseCount))
	t.num("Avg speed", f2(r.AvgSpeedKmh), "km/h")
	t.num("Avg moving speed", f2(r.AvgMovingSpeedKmh), "km/h")
}

// writeSplits renders the per-kilometer table through a grid, so its columns are
// sized to the data: an index past 99 and a duration past nine characters both
// used to break the hand-tuned format this replaces.
func writeSplits(w io.Writer, st style, r stats.Result) {
	writeSection(w, st, "Kilometer splits")
	g := &grid{head: []string{"km", "dist (km)", "time", "speed (km/h)"}}
	for _, s := range r.Splits {
		g.add(
			strconv.Itoa(s.Index),
			f2(s.DistanceKm),
			formatDuration(s.Duration),
			f2(s.SpeedKmh),
		)
	}
	g.flush(w, st)
}

// f2 formats a float to the two decimals every rate and distance in the report
// uses.
func f2(v float64) string { return fmt.Sprintf("%.2f", v) }

// formatDuration renders a duration as a compact H/M/S string (rounded to the
// nearest second).
func formatDuration(d time.Duration) string {
	total := int(d.Round(time.Second) / time.Second)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
