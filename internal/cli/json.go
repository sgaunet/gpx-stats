package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/sgaunet/gpx-stats/internal/stats"
)

// jsonActivity is the wire form of the activity's identity. It is always
// present as an object; each member is a pointer so an absent field serializes
// as null rather than as an empty string that could be mistaken for a real one.
type jsonActivity struct {
	Creator          *string `json:"creator"`
	Name             *string `json:"name"`
	Type             *string `json:"type"`
	MetadataRFC3339  *string `json:"metadataRfc3339"`
	StartTimeRFC3339 *string `json:"startRfc3339"`
	EndTimeRFC3339   *string `json:"endRfc3339"`
}

// jsonResult is the wire representation of a Result. It matches
// contracts/stats.schema.json: camelCase keys, durations in seconds, and
// pointers so unavailable metrics serialize as null rather than zero.
type jsonResult struct {
	Activity             jsonActivity `json:"activity"`
	TotalDistanceKm      float64      `json:"totalDistanceKm"`
	AscendingElevationM  *float64     `json:"ascendingElevationM"`
	DescendingElevationM *float64     `json:"descendingElevationM"`
	EffortKmClimb        *float64     `json:"effortKmClimb"`
	EffortKmClimbDescent *float64     `json:"effortKmClimbDescent"`

	HasElevation      bool        `json:"hasElevation"`
	TotalTimeSeconds  *float64    `json:"totalTimeSeconds"`
	MovingTimeSeconds *float64    `json:"movingTimeSeconds"`
	PauseTimeSeconds  *float64    `json:"pauseTimeSeconds"`
	PauseCount        *int        `json:"pauseCount"`
	HasTimes          bool        `json:"hasTimes"`
	AvgSpeedKmh       *float64    `json:"avgSpeedKmh"`
	AvgMovingSpeedKmh *float64    `json:"avgMovingSpeedKmh"`
	PointCount        int         `json:"pointCount"`
	SegmentCount      int         `json:"segmentCount"`
	Splits            []jsonSplit `json:"splits"`
	Pauses            []jsonPause `json:"pauses"`
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

// optString returns a pointer to s, or nil when s is empty, so the encoder
// distinguishes "absent" from "present but blank".
func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toActivityJSON(a stats.Activity) jsonActivity {
	j := jsonActivity{
		Creator: optString(a.Creator),
		Name:    optString(a.Name),
		Type:    optString(a.Type),
	}
	if a.HasMetadataTime {
		j.MetadataRFC3339 = optString(a.MetadataTime.Format(time.RFC3339))
	}
	if a.HasStartEnd {
		j.StartTimeRFC3339 = optString(a.Start.Format(time.RFC3339))
		j.EndTimeRFC3339 = optString(a.End.Format(time.RFC3339))
	}
	return j
}

func toJSON(r stats.Result) jsonResult {
	j := jsonResult{
		Activity:        toActivityJSON(r.Activity),
		TotalDistanceKm: round3(r.TotalDistanceKm),
		HasElevation:    r.HasElevation,
		HasTimes:        r.HasTimes,
		PointCount:      r.PointCount,
		SegmentCount:    r.SegmentCount,
		Splits:          []jsonSplit{},
		Pauses:          []jsonPause{},
	}
	if r.HasElevation {
		asc := round3(r.AscendingElevationM)
		desc := round3(r.DescendingElevationM)
		climb := round3(r.EffortKmClimb)
		both := round3(r.EffortKmClimbDescent)
		j.AscendingElevationM = &asc
		j.DescendingElevationM = &desc
		j.EffortKmClimb = &climb
		j.EffortKmClimbDescent = &both
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
