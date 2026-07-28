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

// TestDescendingElevation mirrors the ascending table: each case is the
// ascending case with its elevation profile reversed in sign, and must yield
// the same magnitude. Loss is always reported positive.
func TestDescendingElevation(t *testing.T) {
	tests := []struct {
		name  string
		pts   []gpx.TrackPoint
		noise float64
		want  float64
	}{
		{"sum of negative deltas, no noise", elePts(112, 107, 102, 100), 0, 12},
		{"noise filters small drops", elePts(112, 107, 102, 100), 3, 10},
		{"ascend then descend", elePts(100, 110, 105), 0, 5},
		{"flat", elePts(100, 100, 100), 0, 0},
		{"steady small steps counted above threshold", elePts(104, 103, 102, 101, 100), 3, 3},
		{"pure climb has no descent", elePts(100, 110, 120), 3, 0},
		{"points without elevation ignored", []gpx.TrackPoint{{}, {}}, 0, 0},
		{"empty", nil, 3, 0},
		{"single point", elePts(100), 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stats.DescendingElevation(tt.pts, tt.noise)
			if math.Abs(got-tt.want) > 1e-6 {
				t.Errorf("DescendingElevation = %g, want %g", got, tt.want)
			}
			if got < 0 {
				t.Errorf("DescendingElevation must be positive, got %g", got)
			}
		})
	}
}

// TestOutAndBackSymmetry pins FR-001b: on a track that returns to its starting
// elevation, the mirrored hysteresis makes gain and loss agree.
func TestOutAndBackSymmetry(t *testing.T) {
	noise := 3.0
	pts := elePts(100, 120, 140, 120, 100)

	gain := stats.AscendingElevation(pts, noise)
	loss := stats.DescendingElevation(pts, noise)

	if math.Abs(gain-loss) > noise {
		t.Errorf("out-and-back gain %g and loss %g differ by more than the %g m threshold",
			gain, loss, noise)
	}
	if gain <= 0 {
		t.Fatalf("fixture should climb, got gain %g", gain)
	}
}
