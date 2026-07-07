package stats_test

import (
	"math"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

func elePts(elevations ...float64) []gpx.TrackPoint {
	pts := make([]gpx.TrackPoint, len(elevations))
	for i, e := range elevations {
		pts[i] = gpx.TrackPoint{HasEle: true, Ele: e}
	}
	return pts
}

func TestAscendingElevation(t *testing.T) {
	tests := []struct {
		name  string
		pts   []gpx.TrackPoint
		noise float64
		want  float64
	}{
		{"sum of positive deltas, no noise", elePts(100, 105, 110, 112), 0, 12},
		{"noise filters small rises", elePts(100, 105, 110, 112), 3, 10},
		{"descend then ascend", elePts(100, 90, 95), 0, 5},
		{"flat", elePts(100, 100, 100), 0, 0},
		{"steady small steps counted above threshold", elePts(100, 101, 102, 103, 104), 3, 3},
		{"points without elevation ignored", []gpx.TrackPoint{{}, {}}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stats.AscendingElevation(tt.pts, tt.noise)
			if math.Abs(got-tt.want) > 1e-6 {
				t.Errorf("AscendingElevation = %g, want %g", got, tt.want)
			}
		})
	}
}
