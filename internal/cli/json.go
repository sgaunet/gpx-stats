package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/sgaunet/gpx-stats/internal/stats"
)

// jsonResult is the wire representation of a Result. It matches
// contracts/stats.schema.json: camelCase keys, durations in seconds, and
// pointers so unavailable metrics serialize as null rather than zero.
type jsonResult struct {
	TotalDistanceKm     float64     `json:"totalDistanceKm"`
	AscendingElevationM *float64    `json:"ascendingElevationM"`
	HasElevation        bool        `json:"hasElevation"`
	TotalTimeSeconds    *float64    `json:"totalTimeSeconds"`
	MovingTimeSeconds   *float64    `json:"movingTimeSeconds"`
	PauseTimeSeconds    *float64    `json:"pauseTimeSeconds"`
	PauseCount          *int        `json:"pauseCount"`
	HasTimes            bool        `json:"hasTimes"`
	AvgSpeedKmh         *float64    `json:"avgSpeedKmh"`
	AvgMovingSpeedKmh   *float64    `json:"avgMovingSpeedKmh"`
	PointCount          int         `json:"pointCount"`
	Splits              []jsonSplit `json:"splits"`
	Pauses              []jsonPause `json:"pauses"`
}

type jsonSplit struct {
	Index           int     `json:"index"`
	DistanceKm      float64 `json:"distanceKm"`
	DurationSeconds float64 `json:"durationSeconds"`
	SpeedKmh        float64 `json:"speedKmh"`
}

type jsonPause struct {
	StartRFC3339    string  `json:"startRfc3339"`
	EndRFC3339      string  `json:"endRfc3339"`
	DurationSeconds float64 `json:"durationSeconds"`
}

// round3 rounds to three decimal places to keep the JSON tidy and stable.
func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

func toJSON(r stats.Result) jsonResult {
	j := jsonResult{
		TotalDistanceKm: round3(r.TotalDistanceKm),
		HasElevation:    r.HasElevation,
		HasTimes:        r.HasTimes,
		PointCount:      r.PointCount,
		Splits:          []jsonSplit{},
		Pauses:          []jsonPause{},
	}
	if r.HasElevation {
		v := round3(r.AscendingElevationM)
		j.AscendingElevationM = &v
	}
	if r.HasTimes {
		tt := round3(r.TotalTime.Seconds())
		mt := round3(r.MovingTime.Seconds())
		pt := round3(r.PauseTime.Seconds())
		pc := r.PauseCount
		as := round3(r.AvgSpeedKmh)
		ams := round3(r.AvgMovingSpeedKmh)
		j.TotalTimeSeconds = &tt
		j.MovingTimeSeconds = &mt
		j.PauseTimeSeconds = &pt
		j.PauseCount = &pc
		j.AvgSpeedKmh = &as
		j.AvgMovingSpeedKmh = &ams
		for _, s := range r.Splits {
			j.Splits = append(j.Splits, jsonSplit{
				Index:           s.Index,
				DistanceKm:      round3(s.DistanceKm),
				DurationSeconds: round3(s.Duration.Seconds()),
				SpeedKmh:        round3(s.SpeedKmh),
			})
		}
		for _, p := range r.Pauses {
			j.Pauses = append(j.Pauses, jsonPause{
				StartRFC3339:    p.Start.Format(time.RFC3339),
				EndRFC3339:      p.End.Format(time.RFC3339),
				DurationSeconds: round3(p.Duration.Seconds()),
			})
		}
	}
	return j
}

// WriteJSON encodes the result as indented JSON to w (FR-023).
func WriteJSON(w io.Writer, r stats.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(toJSON(r)); err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}
	return nil
}
