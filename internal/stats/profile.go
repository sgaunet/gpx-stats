package stats

import "github.com/sgaunet/gpx-stats/internal/gpx"

// ElevationProfile is an elevation series resampled onto a uniform grid of an
// x-axis quantity (horizontal distance or elapsed time), so it can be plotted
// against evenly-spaced positions without horizontal distortion. The x quantity
// runs linearly from 0 to XMax across len(Elevations) samples.
//
// Charting libraries used by the CLI and web transports plot a bare y-series
// against equally-spaced slots; resampling here is what makes their x-axis
// honest (track points are not evenly spaced in distance, and may be uneven in
// time across pauses).
type ElevationProfile struct {
	Elevations []float64 // resampled elevation, metres
	XMax       float64   // kilometres for distance profiles, seconds for time profiles
}

// ElevationOverDistance resamples the track's elevation onto `samples` points
// evenly spaced by cumulative horizontal distance (kilometres). The second
// return value is false when the track has no usable elevation series (fewer
// than two elevation points) or never moves (zero distance span).
func ElevationOverDistance(track gpx.Track, samples int) (ElevationProfile, bool) {
	if !track.HasElevation {
		return ElevationProfile{}, false
	}
	xs, ys := elevationSeries(track, false)
	return buildProfile(xs, ys, samples)
}

// ElevationOverTime resamples the track's elevation onto `samples` points evenly
// spaced by elapsed time (seconds from the first sample). The second return
// value is false when the track lacks timestamps or elevation, has fewer than
// two usable samples, or spans zero elapsed time.
func ElevationOverTime(track gpx.Track, samples int) (ElevationProfile, bool) {
	if !track.HasElevation || !track.HasTimes {
		return ElevationProfile{}, false
	}
	xs, ys := elevationSeries(track, true)
	return buildProfile(xs, ys, samples)
}

// elevationSeries walks the track once and returns paired (x, elevation) values
// for every point that carries an elevation (and a timestamp, when byTime).
// Cumulative distance accrues across all points so the x values stay correct
// even if an occasional point lacks elevation. xs is monotonic non-decreasing
// by construction (distance never decreases; timestamps are ordered).
func elevationSeries(track gpx.Track, byTime bool) (xs, ys []float64) {
	pts := track.Points
	if len(pts) == 0 {
		return nil, nil
	}
	start := pts[0].Time
	var cumKm float64
	for i, p := range pts {
		// Skipping the gap keeps this axis identical to the reported total
		// distance, so the chart and the statistic never disagree.
		if i > 0 && gpx.SameSegment(pts[i-1], p) {
			cumKm += haversineKm(pts[i-1].Lat, pts[i-1].Lon, p.Lat, p.Lon)
		}
		if !p.HasEle {
			continue
		}
		if byTime {
			if !p.HasTime {
				continue
			}
			xs = append(xs, p.Time.Sub(start).Seconds())
		} else {
			xs = append(xs, cumKm)
		}
		ys = append(ys, p.Ele)
	}
	return xs, ys
}

// buildProfile resamples (xs, ys) onto `samples` uniform points and reports the
// x span. It fails (ok=false) when there is too little data or the span is not
// positive, in which case the caller skips the chart.
func buildProfile(xs, ys []float64, samples int) (ElevationProfile, bool) {
	if len(xs) < 2 || samples < 2 {
		return ElevationProfile{}, false
	}
	span := xs[len(xs)-1] - xs[0]
	if span <= 0 {
		return ElevationProfile{}, false
	}
	return ElevationProfile{
		Elevations: resampleUniform(xs, ys, samples),
		XMax:       span,
	}, true
}

// resampleUniform samples ys at n points evenly spaced across [xs[0], xs[last]]
// using linear interpolation. xs must be non-decreasing, and xs/ys must have the
// same length >= 2 with n >= 2. Duplicate x values are handled by falling back
// to the left sample.
func resampleUniform(xs, ys []float64, n int) []float64 {
	out := make([]float64, n)
	x0 := xs[0]
	span := xs[len(xs)-1] - x0
	j := 0
	for i := range n {
		t := x0 + span*float64(i)/float64(n-1)
		for j < len(xs)-2 && xs[j+1] < t {
			j++
		}
		xa, xb := xs[j], xs[j+1]
		if xb == xa {
			out[i] = ys[j]
			continue
		}
		frac := (t - xa) / (xb - xa)
		out[i] = ys[j] + frac*(ys[j+1]-ys[j])
	}
	return out
}
