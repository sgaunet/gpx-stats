package web_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/web"
)

func TestBuildScaleBar(t *testing.T) {
	for _, tc := range []struct {
		name     string
		km       float64
		segments int
		ticks    []string
	}{
		// 18.85 km: steps of 5, the last segment short. Every segment is
		// labelled at its start.
		{"hike", 18.85, 4, []string{"0", "5", "10", "15"}},
		// A whole number of steps: no short segment.
		{"exact", 10, 5, []string{"0", "2", "4", "6", "8"}},
		// 0.16 km: the 0.01 km remainder is under a quarter step, so it is
		// folded into the last full segment rather than drawn as a sliver.
		{"short", 0.16, 3, []string{"0", "0.05", "0.1"}},
		// 10.15 km: same fold, at a larger scale.
		{"fold", 10.15, 5, []string{"0", "2", "4", "6", "8"}},
		// A remainder of 0.0001 km is folded into the last full segment.
		{"sliver", 20.0001, 4, []string{"0", "5", "10", "15"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bar := web.BuildScaleBar(tc.km)
			if !bar.Show {
				t.Fatalf("scale bar hidden for %v km", tc.km)
			}
			if len(bar.Segments) != tc.segments {
				t.Errorf("segments = %d, want %d", len(bar.Segments), tc.segments)
			}
			var total float64
			for i, s := range bar.Segments {
				w, err := strconv.ParseFloat(s.Width, 64)
				if err != nil {
					t.Fatalf("segment %d width %q: %v", i, s.Width, err)
				}
				total += w
				if s.Filled != (i%2 == 0) {
					t.Errorf("segment %d filled = %v; segments must alternate starting filled", i, s.Filled)
				}
			}
			if math.Abs(total-100) > 0.01 {
				t.Errorf("segment widths sum to %v%%, want 100%%", total)
			}
			if len(bar.Ticks) != len(tc.ticks) {
				t.Fatalf("ticks = %+v, want labels %v", bar.Ticks, tc.ticks)
			}
			for i, want := range tc.ticks {
				if bar.Ticks[i].Label != want {
					t.Errorf("tick %d = %q, want %q", i, bar.Ticks[i].Label, want)
				}
				wantUnit := ""
				if i == len(tc.ticks)-1 {
					wantUnit = "km"
				}
				if bar.Ticks[i].Unit != wantUnit {
					t.Errorf("tick %d unit = %q, want %q (only the last tick carries it)", i, bar.Ticks[i].Unit, wantUnit)
				}
			}
		})
	}
}

func TestBuildScaleBarHiddenWithoutDistance(t *testing.T) {
	for _, km := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if web.BuildScaleBar(km).Show {
			t.Errorf("scale bar shown for %v km", km)
		}
	}
}
