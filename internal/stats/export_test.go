package stats

// Exported aliases so black-box (stats_test) tests can exercise the unexported
// helpers directly. Compiled only during testing.
var (
	HaversineKm          = haversineKm
	AscendingElevation   = ascendingElevation
	DescendingElevation  = descendingElevation
	EffortKmClimb        = effortKmClimb
	EffortKmClimbDescent = effortKmClimbDescent
	DetectPauses         = detectPauses
	KilometerSplits      = kilometerSplits
)
