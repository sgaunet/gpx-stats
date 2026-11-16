package web

import (
	"math"
	"strconv"
)

// scaleBar draws the activity's total distance the way a map sheet draws its
// scale: alternating filled and open segments of a round step, with a label at
// each step. It is presentation only. It divides the total distance that
// stats.Compute already produced and never computes a distance of its own.
type scaleBar struct {
	// Show is false when there is no distance to draw.
	Show bool
	// Segments are the bar's pieces, left to right, alternating fill. The
	// last one is shorter when the distance is not a whole number of steps.
	Segments []scaleSegment
	// Ticks label the start of every segment; the last one carries the unit.
	// The bar's far end is not labelled: the figure printed directly above
	// it already states the total.
	Ticks []scaleTick
}

type scaleSegment struct {
	// Width is the segment's share of the bar, in percent, preformatted.
	Width  string
	Filled bool
}

type scaleTick struct {
	// Left is the tick's offset along the bar, in percent, preformatted.
	Left  string
	Label string
	// Unit is set on the last tick only ("km").
	Unit string
}

// minScaleSteps is the fewest whole steps the bar is divided into; the
// 1-2-5 series then keeps the count between three and about seven.
const minScaleSteps = 3

// minRestSteps is the shortest a final, partial segment may be, in steps. A
// shorter remainder is folded into the segment before it: a sliver would only
// crowd its label against the bar's end.
const minRestSteps = 0.25

// buildScaleBar lays out the scale bar for a distance in kilometres.
func buildScaleBar(km float64) scaleBar {
	if !(km > 0) || math.IsInf(km, 0) {
		return scaleBar{}
	}
	step := niceStep(km / minScaleSteps)
	full := int(math.Floor(km/step + 1e-9))
	rest := km - float64(full)*step

	widths := make([]float64, full, full+1)
	for i := range widths {
		widths[i] = step
	}
	switch {
	case rest <= km*1e-9:
		// A whole number of steps.
	case rest < minRestSteps*step:
		widths[full-1] += rest
	default:
		widths = append(widths, rest)
	}

	bar := scaleBar{Show: true}
	for i, w := range widths {
		start := roundStep(float64(i) * step)
		bar.Segments = append(bar.Segments, scaleSegment{
			Width:  percent(w / km),
			Filled: i%2 == 0,
		})
		bar.Ticks = append(bar.Ticks, scaleTick{
			Left:  percent(start / km),
			Label: strconv.FormatFloat(start, 'f', -1, 64),
		})
	}
	bar.Ticks[len(bar.Ticks)-1].Unit = "km"
	return bar
}

// niceStep returns the largest step from the 1-2-5 series that does not exceed
// target.
func niceStep(target float64) float64 {
	mag := math.Pow(10, math.Floor(math.Log10(target)))
	for _, m := range []float64{5, 2, 1} {
		if m*mag <= target {
			return roundStep(m * mag)
		}
	}
	return roundStep(mag)
}

// roundStep trims floating-point dust (0.15000000000000002) from multiples of a
// decimal step, so labels and loop bounds stay exact.
func roundStep(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

// percent formats a 0-1 share as a percentage for an inline style, to three
// decimals with trailing zeros dropped ("0", "26.525").
func percent(share float64) string {
	return strconv.FormatFloat(math.Round(share*1e5)/1e3, 'f', -1, 64)
}
