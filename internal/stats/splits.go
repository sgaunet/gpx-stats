package stats

import (
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
)

// splitTargetKm is the length of a full kilometer split.
const splitTargetKm = 1.0

// kilometerSplits divides the activity into 1 km splits, apportioning time
// linearly by distance when a segment crosses a kilometer boundary. The caller
// must ensure the track has timestamps. The final split may be partial.
func kilometerSplits(points []gpx.TrackPoint) []KmSplit {
	var splits []KmSplit
	idx := 1
	var splitDist float64
	var splitDur time.Duration

	closeSplit := func(dist float64, dur time.Duration) {
		speed := 0.0
		if dur > 0 {
			speed = dist / dur.Hours()
		}
		splits = append(splits, KmSplit{Index: idx, DistanceKm: dist, Duration: dur, SpeedKmh: speed})
		idx++
	}

	for i := 1; i < len(points); i++ {
		p0, p1 := points[i-1], points[i]
		if !p0.HasTime || !p1.HasTime {
			continue
		}
		dt := p1.Time.Sub(p0.Time)
		if dt < 0 {
			continue
		}
		if !gpx.SameSegment(p0, p1) {
			// Recording was interrupted: no distance accrues, but the elapsed
			// time still belongs to the kilometer that was in progress — the
			// same treatment a standstill already receives below.
			splitDur += dt
			continue
		}
		segDist := haversineKm(p0.Lat, p0.Lon, p1.Lat, p1.Lon)
		if segDist == 0 {
			// Stationary segment: no distance, but its elapsed time still
			// belongs to the current kilometer.
			splitDur += dt
			continue
		}
		for segDist > 0 {
			remaining := splitTargetKm - splitDist
			if segDist < remaining {
				splitDist += segDist
				splitDur += dt
				break
			}
			// The segment reaches (or passes) the kilometer boundary: take the
			// fraction needed to close this split, then carry the remainder.
			frac := remaining / segDist
			boundaryDur := time.Duration(float64(dt) * frac)
			closeSplit(splitTargetKm, splitDur+boundaryDur)
			dt -= boundaryDur
			segDist -= remaining
			splitDist = 0
			splitDur = 0
		}
	}
	if splitDist > 0 {
		closeSplit(splitDist, splitDur)
	}
	return splits
}
