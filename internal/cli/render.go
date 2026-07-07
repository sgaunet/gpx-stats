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
	fmt.Fprintf(w, "Points:            %d\n", r.PointCount)
	fmt.Fprintf(w, "Total distance:    %.2f km\n", r.TotalDistanceKm)

	if r.HasElevation {
		fmt.Fprintf(w, "Ascending elev.:   %.0f m\n", r.AscendingElevationM)
	} else {
		fmt.Fprintln(w, "Ascending elev.:   unavailable (no elevation data)")
	}

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
