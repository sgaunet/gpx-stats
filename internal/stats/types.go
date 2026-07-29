// Package stats computes activity statistics from a parsed GPX track. It is a
// pure domain package: it does not import transport (CLI/HTTP) or storage
// packages, so the same computation backs every interface.
package stats

import "time"

// Activity is the descriptive identity of the analysed file: what recorded it,
// what it is called, and when it happened.
//
// These are not statistics. They are carried here so that every transport
// renders the same identity from the same payload, exactly as it already does
// for the numbers — a transport must never reach back into the parsed track to
// read them itself, or the interfaces can drift apart.
type Activity struct {
	// Creator, Name and Type are copied from the document. Empty means the
	// document did not carry them.
	Creator string
	Name    string
	Type    string

	// MetadataTime is when the FILE was written, which many tools set to the
	// export time. It is deliberately not called the activity date: that is
	// Start, below.
	MetadataTime    time.Time
	HasMetadataTime bool

	// Start and End are the timestamps of the first and last point carrying
	// one. They are available whenever any point is timestamped, which is a
	// weaker condition than Result.HasTimes.
	Start       time.Time
	End         time.Time
	HasStartEnd bool
}

// Result is the full set of statistics computed for an activity. Optional
// metrics use the Has* flags so "unavailable" is distinct from a real zero.
type Result struct {
	// Activity is the file's identity rather than a measurement of it.
	Activity Activity

	TotalDistanceKm float64
	// AscendingElevationM, DescendingElevationM and the two effort figures are
	// all gated by HasElevation: when it is false they stay zero and callers
	// must render them as unavailable rather than as a real zero.
	AscendingElevationM  float64
	DescendingElevationM float64
	// EffortKmClimb is distance + D+/100; EffortKmClimbDescent adds D-/300.
	// Two opinionated conventions, both reported (never one preferred).
	EffortKmClimb        float64
	EffortKmClimbDescent float64
	HasElevation         bool
	TotalTime            time.Duration
	MovingTime           time.Duration
	PauseTime            time.Duration
	PauseCount           int
	HasTimes             bool
	AvgSpeedKmh          float64
	AvgMovingSpeedKmh    float64

	// The four effort rates are each convention's effort kilometers per hour,
	// over elapsed time (EffortSpeedKmh*) and over moving time
	// (EffortMovingSpeedKmh*), mirroring AvgSpeedKmh / AvgMovingSpeedKmh. They
	// are the only metrics here gated by HasElevation *and* HasTimes: without
	// either they stay zero and callers must render them as unavailable.
	EffortSpeedKmhClimb              float64
	EffortSpeedKmhClimbDescent       float64
	EffortMovingSpeedKmhClimb        float64
	EffortMovingSpeedKmhClimbDescent float64

	Splits     []KmSplit
	Pauses     []Pause
	PointCount int

	// SegmentCount is the number of recorded <trkseg> runs the activity is made
	// of. More than one means recording was interrupted, so the distance
	// reported here is smaller than a tool that joins the segments would show.
	// That is worth telling the reader, so transports surface it when it is
	// greater than 1.
	SegmentCount int
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
