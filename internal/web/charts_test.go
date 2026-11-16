package web_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
	"github.com/sgaunet/gpx-stats/internal/web"
)

func TestElevationVsDistanceSVG(t *testing.T) {
	track := gpx.Track{
		HasElevation: true,
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.0, Ele: 100, HasEle: true},
			{Lat: 45, Lon: 6.001, Ele: 110, HasEle: true},
			{Lat: 45, Lon: 6.002, Ele: 108, HasEle: true},
		},
	}
	c, err := web.ElevationVsDistanceChart(track)
	if err != nil {
		t.Fatalf("ElevationVsDistanceChart: %v", err)
	}
	assertBothSizes(t, c)
}

func TestElevationVsDistanceSVGSkippedWithoutElevation(t *testing.T) {
	track := gpx.Track{HasElevation: false, Points: []gpx.TrackPoint{{Lat: 45, Lon: 6}}}
	c, err := web.ElevationVsDistanceChart(track)
	if err != nil {
		t.Fatalf("ElevationVsDistanceChart: %v", err)
	}
	if c != (web.ChartSVG{}) {
		t.Errorf("expected no chart when elevation is unavailable")
	}
}

func TestElevationVsTimeSVG(t *testing.T) {
	base := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	track := gpx.Track{
		HasElevation: true,
		HasTimes:     true,
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.0, Ele: 100, HasEle: true, Time: base, HasTime: true},
			{Lat: 45, Lon: 6.001, Ele: 110, HasEle: true, Time: base.Add(time.Minute), HasTime: true},
			{Lat: 45, Lon: 6.002, Ele: 108, HasEle: true, Time: base.Add(2 * time.Minute), HasTime: true},
		},
	}
	c, err := web.ElevationVsTimeChart(track)
	if err != nil {
		t.Fatalf("ElevationVsTimeChart: %v", err)
	}
	assertBothSizes(t, c)
}

func TestElevationVsTimeSVGSkippedWithoutTimes(t *testing.T) {
	track := gpx.Track{
		HasElevation: true,
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.0, Ele: 100, HasEle: true},
			{Lat: 45, Lon: 6.001, Ele: 110, HasEle: true},
		},
	}
	c, err := web.ElevationVsTimeChart(track)
	if err != nil {
		t.Fatalf("ElevationVsTimeChart: %v", err)
	}
	if c != (web.ChartSVG{}) {
		t.Errorf("expected no chart when timestamps are unavailable")
	}
}

func TestSpeedSVG(t *testing.T) {
	res := stats.Result{
		HasTimes: true,
		Splits: []stats.KmSplit{
			{Index: 1, DistanceKm: 1, Duration: 6 * time.Minute, SpeedKmh: 10},
			{Index: 2, DistanceKm: 1, Duration: 5 * time.Minute, SpeedKmh: 12},
		},
	}
	c, err := web.SpeedChart(res)
	if err != nil {
		t.Fatalf("SpeedChart: %v", err)
	}
	assertBothSizes(t, c)
}

func TestSpeedSVGSkippedWithoutSplits(t *testing.T) {
	c, err := web.SpeedChart(stats.Result{HasTimes: false})
	if err != nil {
		t.Fatalf("SpeedChart: %v", err)
	}
	if c != (web.ChartSVG{}) {
		t.Errorf("expected no chart when there are no splits")
	}
}

func TestSVGFragmentStripsProlog(t *testing.T) {
	in := []byte(`<?xml version="1.0"?>` + "\n" + `<svg xmlns="..."></svg>`)
	out := web.SVGFragment(in)
	if !bytes.HasPrefix(out, []byte("<svg")) {
		t.Errorf("fragment should start with <svg, got %q", out)
	}
}

// assertBothSizes checks a chart was drawn at both layout sizes, themed through
// CSS custom properties, with no sentinel colour left behind.
func assertBothSizes(t *testing.T, c web.ChartSVG) {
	t.Helper()
	for name, svg := range map[string]string{"wide": string(c.Wide), "compact": string(c.Compact)} {
		if !strings.HasPrefix(svg, "<svg") {
			t.Errorf("%s: expected an inline SVG, got %.40q", name, svg)
		}
		if strings.Contains(svg, "rgb(1,2,") || strings.Contains(svg, "rgba(1,2,") {
			t.Errorf("%s: a sentinel colour survived theming", name)
		}
		if !strings.Contains(svg, "var(--chart-series)") {
			t.Errorf("%s: the series is not themed through --chart-series", name)
		}
		if strings.Contains(svg, "Roboto") {
			t.Errorf("%s: the library font leaked; text should use --chart-font", name)
		}
	}
}

// TestThemeSVGMapsEverySentinel pins the sentinel-to-token table, including the
// translucent rgba() form the library uses for the area under a line.
func TestThemeSVGMapsEverySentinel(t *testing.T) {
	in := `<text style="fill:rgb(1,2,3);font-family:'Roboto Medium',sans-serif">` +
		`<path style="stroke:rgb(1,2,4)"/><path style="stroke:rgb(1,2,5)"/>` +
		`<path style="stroke:rgb(1,2,6);fill:rgba(1,2,6,0.275)"/>`
	got := string(web.ThemeSVG([]byte(in)))
	for _, want := range []string{
		"fill:var(--chart-text)",
		"font-family:var(--chart-font)",
		"stroke:var(--chart-axis)",
		"stroke:var(--chart-grid)",
		"stroke:var(--chart-series)",
		"fill:var(--chart-area)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("themed SVG missing %q in %s", want, got)
		}
	}
	// A colour that merely resembles a sentinel is left alone.
	if out := string(web.ThemeSVG([]byte("fill:rgb(1,2,7)"))); out != "fill:rgb(1,2,7)" {
		t.Errorf("non-sentinel colour rewritten to %q", out)
	}
}

func TestFormatAxisDuration(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		want string
	}{
		{0, "0"},
		{29 * time.Second, "0"},
		{45 * time.Minute, "45m"},
		{2*time.Hour + 8*time.Minute + 16*time.Second, "2h08"},
		{12*time.Hour + 53*time.Minute + 34*time.Second, "12h54"},
	} {
		if got := web.FormatAxisDuration(tc.in); got != tc.want {
			t.Errorf("FormatAxisDuration(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestBarShare: bars never grow into slabs on a short activity, and keep a gap
// between them on a long one.
func TestBarShare(t *testing.T) {
	if got := web.BarShare(1, 960); got*910 > 40.01 {
		t.Errorf("a single split's bar is %.0fpx wide, want at most 40px", got*910)
	}
	if got := web.BarShare(19, 960); got != 0.7 {
		t.Errorf("BarShare(19, 960) = %v, want the 0.7 cap", got)
	}
}
