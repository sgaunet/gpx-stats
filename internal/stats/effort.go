package stats

import "time"

// Effort kilometers ("km-effort") express a hilly route as the flat distance
// that would cost the same effort. Two opinionated conventions are in common
// use and both are reported; neither is treated as the correct one.
const (
	// metersOfAscentPerKm is how much climbing counts as one extra kilometer.
	metersOfAscentPerKm = 100
	// metersOfDescentPerKm is the descent equivalent, deliberately cheaper.
	metersOfDescentPerKm = 300
)

// effortKmClimb applies the climb-only convention:
//
//	distance + D+/100
func effortKmClimb(distanceKm, ascentM float64) float64 {
	return distanceKm + ascentM/metersOfAscentPerKm
}

// effortKmClimbDescent applies the convention that also charges for descent:
//
//	distance + D+/100 + D-/300
func effortKmClimbDescent(distanceKm, ascentM, descentM float64) float64 {
	return effortKmClimb(distanceKm, ascentM) + descentM/metersOfDescentPerKm
}

// effortSpeedKmh expresses effort kilometers as a rate over d. A non-positive
// duration yields 0; it is the caller's availability gate, never this value,
// that tells a reader the figure is absent rather than genuinely zero.
func effortSpeedKmh(effortKm float64, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return effortKm / d.Hours()
}
