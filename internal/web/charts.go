package web

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/go-analyze/charts"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

const (
	chartWidth  = 720
	chartHeight = 320

	// elevationSamples is the resampling resolution for elevation profiles: dense
	// enough for a smooth SVG curve. elevationLabelCount thins the x-axis tick
	// labels so they do not collide.
	elevationSamples    = 200
	elevationLabelCount = 7
)

// elevationVsDistanceSVG renders the elevation-over-distance profile as SVG bytes,
// with the x-axis labelled in kilometres. It returns (nil, nil) when the track has
// no usable elevation series (the chart is simply skipped).
func elevationVsDistanceSVG(track gpx.Track) ([]byte, error) {
	prof, ok := stats.ElevationOverDistance(track, elevationSamples)
	if !ok {
		return nil, nil
	}
	prec := kmLabelPrecision(prof.XMax)
	labels := axisLabels(prof.XMax, len(prof.Elevations), func(km float64) string {
		return strconv.FormatFloat(km, 'f', prec, 64)
	})
	return renderElevation(prof.Elevations, "Elevation vs. distance (m)", labels)
}

// elevationVsTimeSVG renders the elevation-over-time profile as SVG bytes, with the
// x-axis labelled in elapsed time. It returns (nil, nil) when the track lacks
// elevation or timestamps (the chart is simply skipped).
func elevationVsTimeSVG(track gpx.Track) ([]byte, error) {
	prof, ok := stats.ElevationOverTime(track, elevationSamples)
	if !ok {
		return nil, nil
	}
	labels := axisLabels(prof.XMax, len(prof.Elevations), func(sec float64) string {
		return formatDuration(time.Duration(sec) * time.Second)
	})
	return renderElevation(prof.Elevations, "Elevation vs. time (m)", labels)
}

// kmLabelPrecision picks a decimal precision for distance-axis labels so that
// consecutive ticks stay distinct whether the track is a few hundred metres or
// tens of kilometres long.
func kmLabelPrecision(xmax float64) int {
	step := xmax / float64(elevationLabelCount-1)
	switch {
	case step < 0.1:
		return 2
	case step < 10:
		return 1
	default:
		return 0
	}
}

// axisLabels builds one label per sample by mapping the uniform x grid
// [0, xmax] through format. LabelCount later thins these for display.
func axisLabels(xmax float64, n int, format func(float64) string) []string {
	labels := make([]string, n)
	for i := range labels {
		labels[i] = format(xmax * float64(i) / float64(n-1))
	}
	return labels
}

// renderElevation draws a resampled elevation series as an SVG line chart with the
// given title and x-axis tick labels.
func renderElevation(ys []float64, title string, labels []string) ([]byte, error) {
	p, err := charts.LineRender(
		[][]float64{ys},
		charts.SVGOutputOptionFunc(),
		charts.DimensionsOptionFunc(chartWidth, chartHeight),
		charts.TitleTextOptionFunc(title),
		charts.XAxisOptionFunc(charts.XAxisOption{Labels: labels, LabelCount: elevationLabelCount}),
	)
	if err != nil {
		return nil, fmt.Errorf("rendering elevation chart: %w", err)
	}
	return p.Bytes()
}

// speedSVG renders per-kilometer speed as SVG bytes, or (nil, nil) when there
// are no splits to plot.
func speedSVG(res stats.Result) ([]byte, error) {
	if !res.HasTimes || len(res.Splits) == 0 {
		return nil, nil
	}
	speeds := make([]float64, 0, len(res.Splits))
	labels := make([]string, 0, len(res.Splits))
	for _, s := range res.Splits {
		speeds = append(speeds, s.SpeedKmh)
		labels = append(labels, strconv.Itoa(s.Index))
	}
	p, err := charts.LineRender(
		[][]float64{speeds},
		charts.SVGOutputOptionFunc(),
		charts.DimensionsOptionFunc(chartWidth, chartHeight),
		charts.TitleTextOptionFunc("Speed per kilometer (km/h)"),
		charts.XAxisLabelsOptionFunc(labels),
	)
	if err != nil {
		return nil, fmt.Errorf("rendering speed chart: %w", err)
	}
	return p.Bytes()
}

// svgFragment trims any XML prolog/doctype so the SVG can be inlined directly
// inside an HTML page. The input is chart-library output, never user content.
func svgFragment(b []byte) []byte {
	if b == nil {
		return nil
	}
	if i := bytes.Index(b, []byte("<svg")); i >= 0 {
		return b[i:]
	}
	return b
}
