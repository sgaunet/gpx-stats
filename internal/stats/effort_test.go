package stats_test

import (
	"math"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/stats"
)

// TestEffortKm covers both opinionated conventions:
//
//	climb           = distance + D+/100
//	climb + descent = distance + D+/100 + D-/300
func TestEffortKm(t *testing.T) {
	tests := []struct {
		name                     string
		distanceKm, ascent, desc float64
		wantClimb, wantBoth      float64
	}{
		// SC-001 reference route: 10 km, 500 m of climb, 300 m of descent.
		{"reference route", 10, 500, 300, 15, 16},
		{"flat route equals distance", 10, 0, 0, 10, 10},
		{"pure climb: both conventions agree", 10, 500, 0, 15, 15},
		{"pure descent: only the second grows", 10, 0, 300, 10, 11},
		{"zero distance", 0, 0, 0, 0, 0},
		{"fractional values", 5.5, 250, 150, 8, 8.5},
		{"descent alone is worth a third of climb", 0, 300, 300, 3, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClimb := stats.EffortKmClimb(tt.distanceKm, tt.ascent)
			if math.Abs(gotClimb-tt.wantClimb) > 1e-9 {
				t.Errorf("EffortKmClimb = %g, want %g", gotClimb, tt.wantClimb)
			}

			gotBoth := stats.EffortKmClimbDescent(tt.distanceKm, tt.ascent, tt.desc)
			if math.Abs(gotBoth-tt.wantBoth) > 1e-9 {
				t.Errorf("EffortKmClimbDescent = %g, want %g", gotBoth, tt.wantBoth)
			}

			// Elevation only ever adds effort (data-model invariant 2).
			if gotClimb < tt.distanceKm {
				t.Errorf("climb effort %g below distance %g", gotClimb, tt.distanceKm)
			}
			if gotBoth < gotClimb {
				t.Errorf("climb+descent effort %g below climb effort %g", gotBoth, gotClimb)
			}
		})
	}
}

// TestEffortDifferenceIsDescentTerm pins data-model invariant 3: the two
// conventions differ by exactly D-/300, whatever the other inputs.
func TestEffortDifferenceIsDescentTerm(t *testing.T) {
	const distanceKm, ascent, desc = 12.34, 678.9, 432.1

	climb := stats.EffortKmClimb(distanceKm, ascent)
	both := stats.EffortKmClimbDescent(distanceKm, ascent, desc)

	if want := desc / 300; math.Abs((both-climb)-want) > 1e-9 {
		t.Errorf("difference = %g, want %g", both-climb, want)
	}
}

// TestEffortSpeedKmh covers the rate: effort kilometers per hour over a given
// duration. A non-positive duration must yield 0 rather than an infinity or a
// NaN — callers gate on HasElevation && HasTimes to say "unavailable".
func TestEffortSpeedKmh(t *testing.T) {
	tests := []struct {
		name     string
		effortKm float64
		d        time.Duration
		want     float64
	}{
		{"one hour is the effort itself", 15, time.Hour, 15},
		{"half an hour doubles the rate", 15, 30 * time.Minute, 30},
		{"two hours halve it", 16, 2 * time.Hour, 8},
		{"partial hour", 12, 45 * time.Minute, 16},
		{"zero effort", 0, time.Hour, 0},
		{"zero duration yields zero, not infinity", 15, 0, 0},
		{"negative duration yields zero", 15, -time.Hour, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stats.EffortSpeedKmh(tt.effortKm, tt.d)
			if math.IsInf(got, 0) || math.IsNaN(got) {
				t.Fatalf("EffortSpeedKmh = %g, want a finite number", got)
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("EffortSpeedKmh = %g, want %g", got, tt.want)
			}
		})
	}
}
