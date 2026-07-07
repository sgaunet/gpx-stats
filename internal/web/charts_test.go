package web_test

import (
	"bytes"
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
	b, err := web.ElevationVsDistanceSVG(track)
	if err != nil {
		t.Fatalf("ElevationVsDistanceSVG: %v", err)
	}
	if !bytes.Contains(b, []byte("<svg")) {
		t.Errorf("expected SVG output, got %d bytes", len(b))
	}
}

func TestElevationVsDistanceSVGSkippedWithoutElevation(t *testing.T) {
	track := gpx.Track{HasElevation: false, Points: []gpx.TrackPoint{{Lat: 45, Lon: 6}}}
	b, err := web.ElevationVsDistanceSVG(track)
	if err != nil {
		t.Fatalf("ElevationVsDistanceSVG: %v", err)
	}
	if b != nil {
		t.Errorf("expected nil SVG when elevation is unavailable")
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
	b, err := web.ElevationVsTimeSVG(track)
	if err != nil {
		t.Fatalf("ElevationVsTimeSVG: %v", err)
	}
	if !bytes.Contains(b, []byte("<svg")) {
		t.Errorf("expected SVG output, got %d bytes", len(b))
	}
}

func TestElevationVsTimeSVGSkippedWithoutTimes(t *testing.T) {
	track := gpx.Track{
		HasElevation: true,
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.0, Ele: 100, HasEle: true},
			{Lat: 45, Lon: 6.001, Ele: 110, HasEle: true},
		},
	}
	b, err := web.ElevationVsTimeSVG(track)
	if err != nil {
		t.Fatalf("ElevationVsTimeSVG: %v", err)
	}
	if b != nil {
		t.Errorf("expected nil SVG when timestamps are unavailable")
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
	b, err := web.SpeedSVG(res)
	if err != nil {
		t.Fatalf("SpeedSVG: %v", err)
	}
	if !bytes.Contains(b, []byte("<svg")) {
		t.Errorf("expected SVG output, got %d bytes", len(b))
	}
}

func TestSpeedSVGSkippedWithoutSplits(t *testing.T) {
	b, err := web.SpeedSVG(stats.Result{HasTimes: false})
	if err != nil {
		t.Fatalf("SpeedSVG: %v", err)
	}
	if b != nil {
		t.Errorf("expected nil SVG when there are no splits")
	}
}

func TestSVGFragmentStripsProlog(t *testing.T) {
	in := []byte(`<?xml version="1.0"?>` + "\n" + `<svg xmlns="..."></svg>`)
	out := web.SVGFragment(in)
	if !bytes.HasPrefix(out, []byte("<svg")) {
		t.Errorf("fragment should start with <svg, got %q", out)
	}
}
