package web

// Exported aliases so black-box (web_test) tests can exercise the unexported
// chart helpers directly. Compiled only during testing.
var (
	ElevationVsDistanceSVG = elevationVsDistanceSVG
	ElevationVsTimeSVG     = elevationVsTimeSVG
	SpeedSVG               = speedSVG
	SVGFragment            = svgFragment

	// BaseLayers exposes the map base-layer table so its invariants can be
	// asserted directly rather than inferred from rendered HTML.
	BaseLayers = baseLayers

	// BuildMapView exposes the map payload builder so the template.JS safety
	// invariant can be asserted on its output directly.
	BuildMapView = buildMapView
)

// BaseLayer is the map base-layer descriptor, aliased for black-box tests.
type BaseLayer = baseLayer

// MapView is the map template payload, aliased for black-box tests.
type MapView = mapView
