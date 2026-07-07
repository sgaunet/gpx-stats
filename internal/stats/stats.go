package stats

import (
	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/gpx"
)

// Compute derives the full statistics Result for a parsed track using the given
// configuration. It is the single entry point shared by every interface, so the
// CLI and web UI always report identical values.
//
// Distance and elevation are computed whenever the data is present; time-based
// metrics (total/moving/pause time, speeds, splits) are computed only when the
// track carries timestamps, and are otherwise left zero with HasTimes=false so
// callers can render them as "unavailable".
func Compute(track gpx.Track, cfg config.Config) Result {
	pts := track.Points
	res := Result{PointCount: len(pts)}

	for i := 1; i < len(pts); i++ {
		res.TotalDistanceKm += haversineKm(pts[i-1].Lat, pts[i-1].Lon, pts[i].Lat, pts[i].Lon)
	}

	res.HasElevation = track.HasElevation
	if track.HasElevation {
		res.AscendingElevationM = ascendingElevation(pts, cfg.ElevationNoiseMeters)
	}

	res.HasTimes = track.HasTimes
	if track.HasTimes && len(pts) >= 2 {
		res.TotalTime = max(pts[len(pts)-1].Time.Sub(pts[0].Time), 0)

		pauses, pauseTime := detectPauses(pts, cfg.StationarySpeedKmh, cfg.MinPauseDuration)
		res.Pauses = pauses
		res.PauseCount = len(pauses)
		res.PauseTime = pauseTime

		res.MovingTime = max(res.TotalTime-pauseTime, 0)

		if res.TotalTime > 0 {
			res.AvgSpeedKmh = res.TotalDistanceKm / res.TotalTime.Hours()
		}
		if res.MovingTime > 0 {
			res.AvgMovingSpeedKmh = res.TotalDistanceKm / res.MovingTime.Hours()
		}
		res.Splits = kilometerSplits(pts)
	}
	return res
}
