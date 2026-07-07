package web

// Exported aliases so black-box (web_test) tests can exercise the unexported
// chart helpers directly. Compiled only during testing.
var (
	ElevationVsDistanceSVG = elevationVsDistanceSVG
	ElevationVsTimeSVG     = elevationVsTimeSVG
	SpeedSVG               = speedSVG
	SVGFragment            = svgFragment
)
