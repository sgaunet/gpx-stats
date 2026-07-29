package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// floatField returns m[key] asserted to float64, failing the test otherwise.
func floatField(t *testing.T, m map[string]any, key string) float64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("field %q is not a number: %v", key, m[key])
	}
	return v
}

// TestWriteJSONShape asserts the JSON output matches the shape declared in
// contracts/stats.schema.json (camelCase keys, seconds, null for unavailable).
func TestWriteJSONShapeFull(t *testing.T) {
	r := stats.Result{
		TotalDistanceKm:     5.5,
		AscendingElevationM: 120,
		HasElevation:        true,
		TotalTime:           100 * time.Second,
		MovingTime:          70 * time.Second,
		PauseTime:           30 * time.Second,
		PauseCount:          1,
		HasTimes:            true,
		AvgSpeedKmh:         2.75,
		AvgMovingSpeedKmh:   3.0,
		PointCount:          8,
		Splits:              []stats.KmSplit{{Index: 1, DistanceKm: 1, Duration: 60 * time.Second, SpeedKmh: 6}},
	}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	for _, k := range []string{"totalDistanceKm", "hasElevation", "hasTimes", "pointCount", "splits"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing required key %q", k)
		}
	}
	if got := floatField(t, m, "totalDistanceKm"); got != 5.5 {
		t.Errorf("totalDistanceKm = %v, want 5.5", got)
	}
	if got := floatField(t, m, "totalTimeSeconds"); got != 100 {
		t.Errorf("totalTimeSeconds = %v, want 100", got)
	}
	if got := floatField(t, m, "ascendingElevationM"); got != 120 {
		t.Errorf("ascendingElevationM = %v, want 120", got)
	}
	splits, ok := m["splits"].([]any)
	if !ok || len(splits) != 1 {
		t.Fatalf("splits should be an array of length 1, got %v", m["splits"])
	}
	sp, ok := splits[0].(map[string]any)
	if !ok {
		t.Fatalf("split[0] is not an object: %v", splits[0])
	}
	for _, k := range []string{"index", "distanceKm", "durationSeconds", "speedKmh"} {
		if _, ok := sp[k]; !ok {
			t.Errorf("split missing key %q", k)
		}
	}
}

func TestWriteJSONUnavailableAreNull(t *testing.T) {
	r := stats.Result{TotalDistanceKm: 1.0, HasElevation: false, HasTimes: false, PointCount: 2}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, k := range []string{
		"ascendingElevationM", "descendingElevationM",
		"effortKmClimb", "effortKmClimbDescent",
		"effortSpeedKmhClimb", "effortSpeedKmhClimbDescent",
		"effortMovingSpeedKmhClimb", "effortMovingSpeedKmhClimbDescent",
		"totalTimeSeconds", "avgSpeedKmh", "pauseCount",
	} {
		if v, ok := m[k]; !ok || v != nil {
			t.Errorf("%q should be null when unavailable, got %v (present=%v)", k, v, ok)
		}
	}
	if arr, ok := m["splits"].([]any); !ok || len(arr) != 0 {
		t.Errorf("splits should be an empty array, got %v", m["splits"])
	}
}

// TestWriteJSONEffort covers the SC-001 reference route through the wire
// format: keys and values as declared in contracts/stats-additions.md.
func TestWriteJSONEffort(t *testing.T) {
	r := stats.Result{
		TotalDistanceKm:      10,
		AscendingElevationM:  500,
		DescendingElevationM: 300,
		EffortKmClimb:        15,
		EffortKmClimbDescent: 16,
		HasElevation:         true,
		PointCount:           8,
	}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	for key, want := range map[string]float64{
		"descendingElevationM": 300,
		"effortKmClimb":        15,
		"effortKmClimbDescent": 16,
	} {
		if got := floatField(t, m, key); got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}

	// The fixture carries no timestamps, so the rates are unavailable even
	// though the elevation half of their gate is satisfied.
	for _, k := range []string{
		"effortSpeedKmhClimb", "effortSpeedKmhClimbDescent",
		"effortMovingSpeedKmhClimb", "effortMovingSpeedKmhClimbDescent",
	} {
		if v, ok := m[k]; !ok || v != nil {
			t.Errorf("%q should be null without timestamps, got %v (present=%v)", k, v, ok)
		}
	}
}

// TestWriteJSONEffortRates walks the reference route over a known elapsed and
// moving time, so each rate key must carry its own distinct value.
func TestWriteJSONEffortRates(t *testing.T) {
	r := stats.Result{
		TotalDistanceKm:      10,
		AscendingElevationM:  500,
		DescendingElevationM: 300,
		EffortKmClimb:        15,
		EffortKmClimbDescent: 16,
		HasElevation:         true,
		HasTimes:             true,
		// One hour elapsed, 30 minutes of it moving.
		TotalTime:                        time.Hour,
		MovingTime:                       30 * time.Minute,
		EffortSpeedKmhClimb:              15,
		EffortSpeedKmhClimbDescent:       16,
		EffortMovingSpeedKmhClimb:        30,
		EffortMovingSpeedKmhClimbDescent: 32,
		PointCount:                       8,
	}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	for key, want := range map[string]float64{
		"effortSpeedKmhClimb":              15,
		"effortSpeedKmhClimbDescent":       16,
		"effortMovingSpeedKmhClimb":        30,
		"effortMovingSpeedKmhClimbDescent": 32,
	} {
		if got := floatField(t, m, key); got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}
}
