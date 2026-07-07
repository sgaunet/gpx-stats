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
func WriteCharts(w io.Writer, track gpx.Track, res stats.Result) {
	fmt.Fprintln(w, "\nCharts")
	fmt.Fprintln(w, "------")

	if prof, ok := stats.ElevationOverDistance(track, asciiWidth); ok {
		fmt.Fprintln(w, "\nElevation vs. distance (m):")
		fmt.Fprintln(w, asciigraph.Plot(prof.Elevations,
			asciigraph.Height(asciiHeight),
			asciigraph.Width(asciiWidth),
			asciigraph.Caption(fmt.Sprintf("distance 0.00 → %.2f km (left → right)", prof.XMax)),
		))
	} else {
		fmt.Fprintln(w, "\nElevation vs. distance: unavailable (no elevation data)")
	}

	if prof, ok := stats.ElevationOverTime(track, asciiWidth); ok {
		fmt.Fprintln(w, "\nElevation vs. time (m):")
		fmt.Fprintln(w, asciigraph.Plot(prof.Elevations,
			asciigraph.Height(asciiHeight),
			asciigraph.Width(asciiWidth),
			asciigraph.Caption(fmt.Sprintf("time 0s → %s elapsed (left → right)",
				formatDuration(time.Duration(prof.XMax)*time.Second))),
		))
	} else {
		fmt.Fprintln(w, "\nElevation vs. time: unavailable (no elevation/timestamps)")
	}

	if res.HasTimes && len(res.Splits) >= 2 {
		speeds := make([]float64, 0, len(res.Splits))
		for _, s := range res.Splits {
			speeds = append(speeds, s.SpeedKmh)
		}
		fmt.Fprintln(w, "\nSpeed per kilometer (km/h):")
		fmt.Fprintln(w, asciigraph.Plot(speeds,
			asciigraph.Height(asciiHeight),
			asciigraph.Width(asciiWidth),
			asciigraph.Caption("speed per kilometer split"),
		))
	} else {
		fmt.Fprintln(w, "\nSpeed chart: unavailable (need at least two kilometer splits)")
	}
}
