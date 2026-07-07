package cli_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

func TestWriteChartsElevationDistance(t *testing.T) {
	track := gpx.Track{
		HasElevation: true,
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.000, Ele: 100, HasEle: true},
			{Lat: 45, Lon: 6.001, Ele: 110, HasEle: true},
			{Lat: 45, Lon: 6.002, Ele: 105, HasEle: true},
			{Lat: 45, Lon: 6.003, Ele: 120, HasEle: true},
		},
	}
	var buf bytes.Buffer
	cli.WriteCharts(&buf, track, stats.Result{})
	out := buf.String()
	if !strings.Contains(out, "Elevation vs. distance (m):") {
		t.Errorf("expected elevation-vs-distance heading:\n%s", out)
	}
	// asciigraph renders axis numbers; the max elevation should appear.
	if !strings.Contains(out, "120") {
		t.Errorf("expected elevation values in the chart:\n%s", out)
	}
	// No timestamps on this track: the time chart is skipped explicitly.
	if !strings.Contains(out, "Elevation vs. time: unavailable") {
		t.Errorf("expected time chart unavailable notice:\n%s", out)
	}
}

func TestWriteChartsElevationTime(t *testing.T) {
	base := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	track := gpx.Track{
		HasElevation: true,
		HasTimes:     true,
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.000, Ele: 100, HasEle: true, Time: base, HasTime: true},
			{Lat: 45, Lon: 6.001, Ele: 110, HasEle: true, Time: base.Add(time.Minute), HasTime: true},
			{Lat: 45, Lon: 6.002, Ele: 120, HasEle: true, Time: base.Add(2 * time.Minute), HasTime: true},
		},
	}
	var buf bytes.Buffer
	cli.WriteCharts(&buf, track, stats.Result{})
	out := buf.String()
	if !strings.Contains(out, "Elevation vs. time (m):") {
		t.Errorf("expected elevation-vs-time heading:\n%s", out)
	}
}

func TestWriteChartsElevationUnavailable(t *testing.T) {
	track := gpx.Track{HasElevation: false, Points: []gpx.TrackPoint{{Lat: 1, Lon: 1}}}
	var buf bytes.Buffer
	cli.WriteCharts(&buf, track, stats.Result{})
	if !strings.Contains(buf.String(), "Elevation vs. distance: unavailable") {
		t.Errorf("expected unavailable notice:\n%s", buf.String())
	}
}

func TestWriteChartsSpeed(t *testing.T) {
	res := stats.Result{
		HasTimes: true,
		Splits: []stats.KmSplit{
			{Index: 1, SpeedKmh: 10, Duration: 6 * time.Minute},
			{Index: 2, SpeedKmh: 12, Duration: 5 * time.Minute},
			{Index: 3, SpeedKmh: 9, Duration: 7 * time.Minute},
		},
	}
	var buf bytes.Buffer
	cli.WriteCharts(&buf, gpx.Track{}, res)
	if !strings.Contains(buf.String(), "Speed per kilometer (km/h):") {
		t.Errorf("expected speed chart heading:\n%s", buf.String())
	}
}

func TestWriteChartsSpeedUnavailable(t *testing.T) {
	// Only one split → not enough to draw a speed chart.
	res := stats.Result{HasTimes: true, Splits: []stats.KmSplit{{Index: 1, SpeedKmh: 10}}}
	var buf bytes.Buffer
	cli.WriteCharts(&buf, gpx.Track{}, res)
	if !strings.Contains(buf.String(), "Speed chart: unavailable") {
		t.Errorf("expected speed unavailable notice:\n%s", buf.String())
	}
}
