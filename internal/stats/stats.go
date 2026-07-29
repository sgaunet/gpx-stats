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
	res := Result{PointCount: len(pts), Activity: activityOf(track)}
	if len(pts) > 0 {
		// Segment indices are dense and non-decreasing, so the last one names
		// the count.
		res.SegmentCount = pts[len(pts)-1].SegmentIndex + 1
	}

	for i := 1; i < len(pts); i++ {
		// Nothing accrues across a segment boundary: recording was interrupted,
		// so the straight line between these two points was never travelled.
		if !gpx.SameSegment(pts[i-1], pts[i]) {
			continue
		}
		res.TotalDistanceKm += haversineKm(pts[i-1].Lat, pts[i-1].Lon, pts[i].Lat, pts[i].Lon)
	}

	res.HasElevation = track.HasElevation
	if track.HasElevation {
		res.AscendingElevationM = ascendingElevation(pts, cfg.ElevationNoiseMeters)
		res.DescendingElevationM = descendingElevation(pts, cfg.ElevationNoiseMeters)
		// Effort kilometers are derived from the same filtered figures that are
		// reported, so a reader can reproduce them from the numbers on screen.
		res.EffortKmClimb = effortKmClimb(res.TotalDistanceKm, res.AscendingElevationM)
		res.EffortKmClimbDescent = effortKmClimbDescent(
			res.TotalDistanceKm, res.AscendingElevationM, res.DescendingElevationM)
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
		// The effort rates need elevation as well as time. The effort totals
		// they divide are already filled in by the elevation block above, so
		// this is the one place the division happens: transports only read it.
		if res.HasElevation {
			res.EffortSpeedKmhClimb = effortSpeedKmh(res.EffortKmClimb, res.TotalTime)
			res.EffortSpeedKmhClimbDescent = effortSpeedKmh(res.EffortKmClimbDescent, res.TotalTime)
			res.EffortMovingSpeedKmhClimb = effortSpeedKmh(res.EffortKmClimb, res.MovingTime)
			res.EffortMovingSpeedKmhClimbDescent = effortSpeedKmh(res.EffortKmClimbDescent, res.MovingTime)
		}
		res.Splits = kilometerSplits(pts)
	}
	return res
}

// activityOf copies the document's descriptive identity and derives the
// activity's start and end from the first and last timestamped point.
//
// Start and End scan for timestamps rather than reading the endpoints
// directly, so a track whose very first or last point lacks one still reports
// when it happened. They are in document order, not sorted: a track with
// out-of-order timestamps is malformed, and silently reordering it here would
// disagree with every other statistic, all of which read the track as recorded.
func activityOf(track gpx.Track) Activity {
	a := Activity{
		Creator:         track.Creator,
		Name:            track.Name,
		Type:            track.Type,
		MetadataTime:    track.MetadataTime,
		HasMetadataTime: track.HasMetadataTime,
	}
	pts := track.Points
	for _, p := range pts {
		if p.HasTime {
			a.Start = p.Time
			a.HasStartEnd = true
			break
		}
	}
	for i := len(pts) - 1; i >= 0; i-- {
		if pts[i].HasTime {
			a.End = pts[i].Time
			break
		}
	}
	return a
}
