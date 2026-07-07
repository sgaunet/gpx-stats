package stats_test

import (
	"math"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/stats"
)

func TestHaversineKm(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		want, tol              float64
	}{
		{"same point", 45, 6, 45, 6, 0, 1e-9},
		{"one degree longitude at equator", 0, 0, 0, 1, 111.19, 0.5},
		{"one degree latitude", 0, 0, 1, 0, 111.19, 0.5},
		{"paris to london", 48.8566, 2.3522, 51.5074, -0.1278, 343.5, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stats.HaversineKm(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(got-tt.want) > tt.tol {
				t.Errorf("HaversineKm = %g, want %g ± %g", got, tt.want, tt.tol)
			}
		})
	}
}

func TestHaversineSymmetric(t *testing.T) {
	a := stats.HaversineKm(45, 6, 46, 7)
	b := stats.HaversineKm(46, 7, 45, 6)
	if math.Abs(a-b) > 1e-9 {
		t.Errorf("distance not symmetric: %g vs %g", a, b)
	}
}
