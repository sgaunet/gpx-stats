package stats

import "github.com/sgaunet/gpx-stats/internal/gpx"

// ascendingElevation returns the cumulative positive elevation gain in meters.
//
// It uses a hysteresis baseline: gain is only counted once elevation rises at
// least noiseMeters above the current baseline (then the baseline moves up);
// when elevation drops below the baseline, the baseline follows it down. This
// captures sustained climbs (even in small steps) while filtering GPS jitter.
// noiseMeters == 0 makes it the plain sum of positive deltas.
//
// The baseline resets at every segment boundary: the elevation difference
// across an interrupted recording was not climbed on this track, and counting
// it would inflate not only the reported gain but both effort-kilometer figures
// derived from it.
func ascendingElevation(points []gpx.TrackPoint, noiseMeters float64) float64 {
	var gain, baseline float64
	haveBaseline := false
	for i, p := range points {
		// Before the HasEle check on purpose, so the reset survives a new
		// segment whose first point happens to carry no elevation.
		if i > 0 && !gpx.SameSegment(points[i-1], p) {
			haveBaseline = false
		}
		if !p.HasEle {
			continue
		}
		if !haveBaseline {
			baseline = p.Ele
			haveBaseline = true
			continue
		}
		switch {
		case p.Ele >= baseline+noiseMeters:
			gain += p.Ele - baseline
			baseline = p.Ele
		case p.Ele < baseline:
			baseline = p.Ele
		}
	}
	return gain
}

// descendingElevation returns the cumulative elevation loss in meters, as a
// positive quantity.
//
// It is the mirror image of ascendingElevation and uses the same hysteresis
// threshold: loss is only counted once elevation falls at least noiseMeters
// below the current baseline (then the baseline moves down); when elevation
// rises above the baseline, the baseline follows it up. Keeping the two
// symmetric is what makes a closed loop report comparable gain and loss
// instead of a jitter-inflated descent. The baseline resets at every segment
// boundary for the same reason it does for ascent.
func descendingElevation(points []gpx.TrackPoint, noiseMeters float64) float64 {
	var loss, baseline float64
	haveBaseline := false
	for i, p := range points {
		if i > 0 && !gpx.SameSegment(points[i-1], p) {
			haveBaseline = false
		}
		if !p.HasEle {
			continue
		}
		if !haveBaseline {
			baseline = p.Ele
			haveBaseline = true
			continue
		}
		switch {
		case p.Ele <= baseline-noiseMeters:
			loss += baseline - p.Ele
			baseline = p.Ele
		case p.Ele > baseline:
			baseline = p.Ele
		}
	}
	return loss
}
