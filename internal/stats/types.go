// Package stats computes activity statistics from a parsed GPX track. It is a
// pure domain package: it does not import transport (CLI/HTTP) or storage
// packages, so the same computation backs every interface.
package stats

import "time"

// Result is the full set of statistics computed for an activity. Optional
// metrics use the Has* flags so "unavailable" is distinct from a real zero.
type Result struct {
	TotalDistanceKm     float64
	AscendingElevationM float64
	HasElevation        bool
	TotalTime           time.Duration
	MovingTime          time.Duration
	PauseTime           time.Duration
	PauseCount          int
	HasTimes            bool
	AvgSpeedKmh         float64
	AvgMovingSpeedKmh   float64
	Splits              []KmSplit
	Pauses              []Pause
	PointCount          int
}

// KmSplit is one kilometer of the activity (the final split may be partial).
type KmSplit struct {
	Index      int
	DistanceKm float64
	Duration   time.Duration
	SpeedKmh   float64
}

// Pause is a detected stationary interval.
type Pause struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
}
