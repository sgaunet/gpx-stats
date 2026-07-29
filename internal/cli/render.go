// Package cli implements the command-line transport for gpx-stats: flag
// parsing, dispatch, and rendering of statistics as text or JSON. It depends on
// the domain packages (gpx, stats, config) but they never depend on it.
package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/sgaunet/gpx-stats/internal/stats"
)

// WriteText renders a Result as human-readable text to w. Unavailable metrics
// are labelled explicitly rather than shown as a misleading zero.
func WriteText(w io.Writer, r stats.Result) {
	fmt.Fprintln(w, "GPX Statistics")
	fmt.Fprintln(w, "==============")
	writeActivity(w, r.Activity)
	fmt.Fprintf(w, "Points:            %d\n", r.PointCount)
	// The segment note only appears when it explains something. A reader
	// comparing this figure against a tool that joins the segments would
	// otherwise have no way to account for the difference.
	if r.SegmentCount > 1 {
		fmt.Fprintf(w, "Total distance:    %.2f km (%d segments; gaps between them are not counted)\n",
			r.TotalDistanceKm, r.SegmentCount)
	} else {
		fmt.Fprintf(w, "Total distance:    %.2f km\n", r.TotalDistanceKm)
	}

	if r.HasElevation {
		fmt.Fprintf(w, "Ascending elev.:   %.0f m\n", r.AscendingElevationM)
		fmt.Fprintf(w, "Descending elev.:  %.0f m\n", r.DescendingElevationM)
	} else {
		fmt.Fprintln(w, "Ascending elev.:   unavailable (no elevation data)")
		fmt.Fprintln(w, "Descending elev.:  unavailable (no elevation data)")
	}

	// Effort kilometers come before the time-based section on purpose: they are
	// derived from distance and elevation only, so a track without timestamps
	// must still report them.
	writeEffort(w, r)

	if !r.HasTimes {
		fmt.Fprintln(w, "Time-based stats:  unavailable (no timestamps)")
		return
	}

	fmt.Fprintf(w, "Total time:        %s\n", formatDuration(r.TotalTime))
	fmt.Fprintf(w, "Moving time:       %s\n", formatDuration(r.MovingTime))
	fmt.Fprintf(w, "Pause time:        %s (%d pauses)\n", formatDuration(r.PauseTime), r.PauseCount)
	fmt.Fprintf(w, "Avg speed:         %.2f km/h\n", r.AvgSpeedKmh)
	fmt.Fprintf(w, "Avg moving speed:  %.2f km/h\n", r.AvgMovingSpeedKmh)

	if len(r.Splits) > 0 {
		fmt.Fprintln(w, "\nKilometer splits:")
		fmt.Fprintln(w, "  km   dist(km)   time      speed(km/h)")
		for _, s := range r.Splits {
			fmt.Fprintf(w, "  %2d   %6.2f     %-8s  %6.2f\n",
				s.Index, s.DistanceKm, formatDuration(s.Duration), s.SpeedKmh)
		}
	}
}

// writeActivity renders whatever the file said about itself, one line per
// field that is actually present.
//
// A file carrying none of it gets a single "unavailable" line rather than six:
// the terminal is the human surface, and six identical negatives would push the
// statistics down the screen to say nothing. The machine-readable output is
// where every field is always present.
func writeActivity(w io.Writer, a stats.Activity) {
	if a.Name == "" && a.Type == "" && a.Creator == "" && !a.HasMetadataTime && !a.HasStartEnd {
		fmt.Fprintln(w, "Activity:          unavailable (no identity metadata in the file)")
		return
	}
	if a.Name != "" {
		fmt.Fprintf(w, "Activity:          %s\n", a.Name)
	}
	if a.Type != "" {
		fmt.Fprintf(w, "Type:              %s\n", a.Type)
	}
	if a.Creator != "" {
		fmt.Fprintf(w, "Recorded by:       %s\n", a.Creator)
	}
	if a.HasStartEnd {
		fmt.Fprintf(w, "Start:             %s\n", a.Start.Format(time.RFC3339))
		fmt.Fprintf(w, "End:               %s\n", a.End.Format(time.RFC3339))
	}
	if a.HasMetadataTime {
		// Labelled "File time" on purpose: it is when the document was written,
		// which exporters routinely set to the moment you downloaded it. Calling
		// it a date would misdate the activity.
		fmt.Fprintf(w, "File time:         %s\n", a.MetadataTime.Format(time.RFC3339))
	}
}

// writeEffort renders the two effort-kilometer conventions, each with its
// per-hour rates and then its legend. Grouping by convention rather than by
// figure is what lets one legend cover a convention's total and both of its
// rates, instead of repeating the equivalence four times.
//
// Their labels are wider than the column used by the statistics above, so they
// form their own aligned block rather than re-aligning lines that must stay
// unchanged. Neither convention is presented as the correct one.
func writeEffort(w io.Writer, r stats.Result) {
	if !r.HasElevation {
		fmt.Fprintln(w, "Effort km:         unavailable (no elevation data)")
		return
	}
	const labelWidth = -37 // widest label, "Moving effort km/h (climb + descent):"
	fmt.Fprintf(w, "%*s %.2f\n", labelWidth, "Effort km (climb):", r.EffortKmClimb)
	// The rates need timestamps; the totals above do not. A track without them
	// keeps its effort kilometers and is accounted for by the "Time-based
	// stats: unavailable" line that follows this block.
	if r.HasTimes {
		fmt.Fprintf(w, "%*s %.2f km/h\n", labelWidth, "Effort km/h (climb):", r.EffortSpeedKmhClimb)
		fmt.Fprintf(w, "%*s %.2f km/h\n", labelWidth, "Moving effort km/h (climb):", r.EffortMovingSpeedKmhClimb)
	}
	fmt.Fprintln(w, "  100 m ascent = 1 km")
	fmt.Fprintf(w, "%*s %.2f\n", labelWidth, "Effort km (climb + descent):", r.EffortKmClimbDescent)
	if r.HasTimes {
		fmt.Fprintf(w, "%*s %.2f km/h\n", labelWidth,
			"Effort km/h (climb + descent):", r.EffortSpeedKmhClimbDescent)
		fmt.Fprintf(w, "%*s %.2f km/h\n", labelWidth,
			"Moving effort km/h (climb + descent):", r.EffortMovingSpeedKmhClimbDescent)
	}
	fmt.Fprintln(w, "  100 m ascent = 1 km, 300 m descent = 1 km")
}

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
