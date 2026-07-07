package stats

// Exported aliases so black-box (stats_test) tests can exercise the unexported
// helpers directly. Compiled only during testing.
var (
	HaversineKm        = haversineKm
	AscendingElevation = ascendingElevation
	DetectPauses       = detectPauses
	KilometerSplits    = kilometerSplits
)
