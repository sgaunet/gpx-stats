package stats

import (
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
)

// detectPauses groups consecutive stationary segments (segment speed at or
// below stationaryKmh) into pauses. A run counts as a pause only if its total
// duration reaches minDuration. Returns the pauses and their total time.
//
// The comparison is inclusive so that a stationaryKmh of 0 stays meaningful: it
// then matches only a true standstill, where consecutive points share the same
// position and the segment speed is exactly zero.
//
// Segments whose endpoints lack timestamps, or whose timestamps are
// non-increasing (duplicate/out-of-order), are ignored rather than treated as
// movement or as a crash.
func detectPauses(points []gpx.TrackPoint, stationaryKmh float64, minDuration time.Duration) ([]Pause, time.Duration) {
	var (
		pauses   []Pause
		total    time.Duration
		runStart time.Time
		runEnd   time.Time
		runDur   time.Duration
		inRun    bool
	)

	flush := func() {
		if inRun && runDur >= minDuration {
			pauses = append(pauses, Pause{Start: runStart, End: runEnd, Duration: runDur})
			total += runDur
		}
		inRun = false
		runDur = 0
	}

	for i := 1; i < len(points); i++ {
		p0, p1 := points[i-1], points[i]
		if !p0.HasTime || !p1.HasTime {
			flush()
			continue
		}
		dt := p1.Time.Sub(p0.Time)
		if dt <= 0 {
			continue
		}
		// Across a segment boundary no distance was travelled that we know of,
		// so the interval reads as stationary and becomes a pause once it is
		// long enough. That is what keeps moving + pause == total: the gap has
		// to land in one of the two, and it was certainly not movement.
		var distKm float64
		if gpx.SameSegment(p0, p1) {
			distKm = haversineKm(p0.Lat, p0.Lon, p1.Lat, p1.Lon)
		}
		speedKmh := distKm / dt.Hours()
		if speedKmh <= stationaryKmh {
			if !inRun {
				inRun = true
				runStart = p0.Time
				runDur = 0
			}
			runDur += dt
			runEnd = p1.Time
		} else {
			flush()
		}
	}
	flush()
	return pauses, total
}
