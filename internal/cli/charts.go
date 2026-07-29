package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/guptarohit/asciigraph"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

const (
	asciiHeight = 12
	asciiWidth  = 60
)

// WriteCharts renders ASCII elevation and speed charts to w. A chart whose data
// is unavailable is skipped with a short note rather than drawn empty.
//
// The plots keep their fixed width: asciigraph lays out its own axis labels, and
// a chart that changes shape with the terminal cannot be compared against one
// pasted into an issue.
func WriteCharts(w io.Writer, track gpx.Track, res stats.Result, opts ...Option) {
	st := styleFor(w, opts)
	writeSection(w, st, "Charts")

	if prof, ok := stats.ElevationOverDistance(track, asciiWidth); ok {
		fmt.Fprintf(w, "\n%s\n", st.bold("Elevation vs. distance (m):"))
		fmt.Fprintln(w, asciigraph.Plot(prof.Elevations,
			asciigraph.Height(asciiHeight),
			asciigraph.Width(asciiWidth),
			asciigraph.Caption(fmt.Sprintf("distance 0.00 → %.2f km (left → right)", prof.XMax)),
		))
	} else {
		fmt.Fprintf(w, "\n%s\n", st.dim("Elevation vs. distance: unavailable (no elevation data)"))
	}

	if prof, ok := stats.ElevationOverTime(track, asciiWidth); ok {
		fmt.Fprintf(w, "\n%s\n", st.bold("Elevation vs. time (m):"))
		fmt.Fprintln(w, asciigraph.Plot(prof.Elevations,
			asciigraph.Height(asciiHeight),
			asciigraph.Width(asciiWidth),
			asciigraph.Caption(fmt.Sprintf("time 0s → %s elapsed (left → right)",
				formatDuration(time.Duration(prof.XMax)*time.Second))),
		))
	} else {
		fmt.Fprintf(w, "\n%s\n", st.dim("Elevation vs. time: unavailable (no elevation/timestamps)"))
	}

	if res.HasTimes && len(res.Splits) >= 2 {
		speeds := make([]float64, 0, len(res.Splits))
		for _, s := range res.Splits {
			speeds = append(speeds, s.SpeedKmh)
		}
		fmt.Fprintf(w, "\n%s\n", st.bold("Speed per kilometer (km/h):"))
		fmt.Fprintln(w, asciigraph.Plot(speeds,
			asciigraph.Height(asciiHeight),
			asciigraph.Width(asciiWidth),
			asciigraph.Caption("speed per kilometer split"),
		))
	} else {
		fmt.Fprintf(w, "\n%s\n", st.dim("Speed chart: unavailable (need at least two kilometer splits)"))
	}
}
