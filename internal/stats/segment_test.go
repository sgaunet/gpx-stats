package stats_test

import (
	"math"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// segPoint builds a track point in an explicit segment. Every field a segment
// test cares about is visible at the call site.
func segPoint(seg int, lon, ele float64, sec int) gpx.TrackPoint {
	return gpx.TrackPoint{
		Lat: 45, Lon: lon, Ele: ele, HasEle: true,
		HasTime: true, Time: computeBase().Add(time.Duration(sec) * time.Second),
		SegmentIndex: seg,
	}
}

// twoSegmentTrack mirrors testdata/two_segments.gpx: three points, a ten-minute
// interruption during which the device moved ~7.8 km, then three more points.
func twoSegmentTrack() gpx.Track {
	return gpx.Track{
		HasElevation: true, HasTimes: true,
		Points: []gpx.TrackPoint{
			segPoint(0, 6.0000, 100, 0),
			segPoint(0, 6.0004, 105, 10),
			segPoint(0, 6.0008, 110, 20),
			segPoint(1, 6.1000, 300, 620),
			segPoint(1, 6.1004, 305, 630),
			segPoint(1, 6.1008, 310, 640),
		},
	}
}

// oneSegmentEquivalent is twoSegmentTrack with the boundary erased: the same
// coordinates and times, all in one segment. It is what the tool reported
// before segment awareness, and it exists so the tests can state the size of
// the phantom distance rather than merely assert the corrected figure.
func oneSegmentEquivalent() gpx.Track {
	tr := twoSegmentTrack()
	for i := range tr.Points {
		tr.Points[i].SegmentIndex = 0
	}
	return tr
}

func TestComputeSkipsDistanceAcrossSegmentBoundary(t *testing.T) {
	cfg := config.Default()
	got := stats.Compute(twoSegmentTrack(), cfg)
	flattened := stats.Compute(oneSegmentEquivalent(), cfg)

	// Two runs of two ~31.45 m steps each.
	const want = 0.1258028724
	if math.Abs(got.TotalDistanceKm-want) > 1e-6 {
		t.Errorf("TotalDistanceKm = %v, want %v", got.TotalDistanceKm, want)
	}
	// The gap is ~7.8 km of ground the device was carried over, not travelled.
	if flattened.TotalDistanceKm < 7 {
		t.Fatalf("fixture is not exercising a large gap: flattened distance %v", flattened.TotalDistanceKm)
	}
	if got.TotalDistanceKm >= flattened.TotalDistanceKm {
		t.Errorf("segment-aware distance %v should be below the flattened %v",
			got.TotalDistanceKm, flattened.TotalDistanceKm)
	}
}

func TestComputeSegmentCount(t *testing.T) {
	tests := []struct {
		name  string
		track gpx.Track
		want  int
	}{
		{"empty track", gpx.Track{}, 0},
		{"single segment", fullTrack(), 1},
		{"two segments", twoSegmentTrack(), 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stats.Compute(tc.track, config.Default()).SegmentCount; got != tc.want {
				t.Errorf("SegmentCount = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestComputeGapBecomesPause(t *testing.T) {
	res := stats.Compute(twoSegmentTrack(), config.Default())

	if res.TotalTime != 640*time.Second {
		t.Errorf("TotalTime = %s, want 10m40s (unchanged by segment awareness)", res.TotalTime)
	}
	if res.PauseCount != 1 {
		t.Fatalf("PauseCount = %d, want 1", res.PauseCount)
	}
	if res.PauseTime != 600*time.Second {
		t.Errorf("PauseTime = %s, want 10m0s", res.PauseTime)
	}
	if res.MovingTime != 40*time.Second {
		t.Errorf("MovingTime = %s, want 40s", res.MovingTime)
	}
	// The guarantee that made "gap becomes pause time" the right call.
	if res.MovingTime+res.PauseTime != res.TotalTime {
		t.Errorf("moving %s + pause %s != total %s", res.MovingTime, res.PauseTime, res.TotalTime)
	}
}

func TestComputeSubThresholdGapStaysMoving(t *testing.T) {
	// A gap shorter than the minimum pause duration is treated exactly like a
	// short standstill: no pause is recorded and its time stays in moving time.
	// One definition of "pause", no special case for segment gaps.
	track := gpx.Track{
		HasElevation: true, HasTimes: true,
		Points: []gpx.TrackPoint{
			segPoint(0, 6.0000, 100, 0),
			segPoint(0, 6.0004, 100, 10),
			segPoint(1, 6.1000, 100, 40), // 30s gap, below the 2m default
			segPoint(1, 6.1004, 100, 50),
		},
	}
	res := stats.Compute(track, config.Default())

	if res.PauseCount != 0 {
		t.Errorf("PauseCount = %d, want 0 for a gap below the minimum pause duration", res.PauseCount)
	}
	if res.MovingTime != res.TotalTime {
		t.Errorf("MovingTime = %s, want the full %s", res.MovingTime, res.TotalTime)
	}
	if res.MovingTime+res.PauseTime != res.TotalTime {
		t.Errorf("moving %s + pause %s != total %s", res.MovingTime, res.PauseTime, res.TotalTime)
	}
}

func TestElevationBaselineResetsAtSegmentBoundary(t *testing.T) {
	cfg := config.Default()
	got := stats.Compute(twoSegmentTrack(), cfg)
	flattened := stats.Compute(oneSegmentEquivalent(), cfg)

	// 10 m climbed in each segment. The 190 m step between them was driven, not
	// climbed, and must not reach either the gain or the effort figures.
	if got.AscendingElevationM != 20 {
		t.Errorf("AscendingElevationM = %v, want 20", got.AscendingElevationM)
	}
	if flattened.AscendingElevationM != 210 {
		t.Errorf("flattened AscendingElevationM = %v, want 210 — fixture no longer proves the point",
			flattened.AscendingElevationM)
	}
	if got.EffortKmClimb >= flattened.EffortKmClimb {
		t.Errorf("effort km should fall with the phantom climb: got %v, flattened %v",
			got.EffortKmClimb, flattened.EffortKmClimb)
	}
}

func TestElevationDescentBaselineResetsAtSegmentBoundary(t *testing.T) {
	// The mirror case: a big drop across the gap must not be counted as descent.
	track := gpx.Track{
		HasElevation: true, HasTimes: true,
		Points: []gpx.TrackPoint{
			segPoint(0, 6.0000, 300, 0),
			segPoint(0, 6.0004, 290, 10),
			segPoint(1, 6.1000, 100, 620),
			segPoint(1, 6.1004, 90, 630),
		},
	}
	if got := stats.Compute(track, config.Default()).DescendingElevationM; got != 20 {
		t.Errorf("DescendingElevationM = %v, want 20 (10 m lost in each segment)", got)
	}
}

func TestElevationBaselineResetSurvivesMissingElevation(t *testing.T) {
	// The reset happens before the has-elevation check, so a new segment whose
	// first point carries no elevation still starts from a fresh baseline.
	pts := []gpx.TrackPoint{
		{Lat: 45, Lon: 6.0000, Ele: 100, HasEle: true, SegmentIndex: 0},
		{Lat: 45, Lon: 6.0004, Ele: 110, HasEle: true, SegmentIndex: 0},
		{Lat: 45, Lon: 6.1000, SegmentIndex: 1}, // no elevation on the first point
		{Lat: 45, Lon: 6.1004, Ele: 500, HasEle: true, SegmentIndex: 1},
		{Lat: 45, Lon: 6.1008, Ele: 510, HasEle: true, SegmentIndex: 1},
	}
	// 10 m in segment 0; in segment 1 the 500 m point becomes the new baseline
	// and 10 m is climbed from it. The 390 m step across the gap is not counted.
	if got := stats.AscendingElevation(pts, 1.0); got != 20 {
		t.Errorf("AscendingElevation = %v, want 20", got)
	}
}

func TestKilometerSplitsCreditGapTimeWithoutDistance(t *testing.T) {
	res := stats.Compute(twoSegmentTrack(), config.Default())

	if len(res.Splits) != 1 {
		t.Fatalf("got %d splits, want 1 (the activity covers well under a kilometer)", len(res.Splits))
	}
	sp := res.Splits[0]
	// The gap contributes its ten minutes but none of its distance, exactly as a
	// standstill would.
	if sp.Duration != res.TotalTime {
		t.Errorf("split duration = %s, want the full %s", sp.Duration, res.TotalTime)
	}
	if math.Abs(sp.DistanceKm-res.TotalDistanceKm) > 1e-9 {
		t.Errorf("split distance = %v, want the total %v", sp.DistanceKm, res.TotalDistanceKm)
	}
}

func TestComputeActivityIdentityIsCarriedThrough(t *testing.T) {
	track := twoSegmentTrack()
	track.Creator = "Garmin Edge 530"
	track.Name = "Morning Leg"
	track.Type = "running"
	track.MetadataTime = computeBase().Add(-time.Minute)
	track.HasMetadataTime = true

	a := stats.Compute(track, config.Default()).Activity

	if a.Creator != "Garmin Edge 530" || a.Name != "Morning Leg" || a.Type != "running" {
		t.Errorf("identity not carried through: %+v", a)
	}
	if !a.HasMetadataTime || !a.MetadataTime.Equal(computeBase().Add(-time.Minute)) {
		t.Errorf("MetadataTime = %v (has=%v), want one minute before the first point",
			a.MetadataTime, a.HasMetadataTime)
	}
	if !a.HasStartEnd {
		t.Fatalf("HasStartEnd = false, want true for a timestamped track")
	}
	// The invariant that makes Start/End trustworthy alongside TotalTime.
	if got := stats.Compute(track, config.Default()).TotalTime; a.End.Sub(a.Start) != got {
		t.Errorf("End-Start = %s, want TotalTime %s", a.End.Sub(a.Start), got)
	}
}

func TestComputeActivityStartEndSkipUntimedEndpoints(t *testing.T) {
	// A track whose first and last points lack timestamps still happened at a
	// knowable time; reading the endpoints blindly would report a zero time.
	base := computeBase()
	track := gpx.Track{
		Points: []gpx.TrackPoint{
			{Lat: 45, Lon: 6.0000},
			{Lat: 45, Lon: 6.0004, HasTime: true, Time: base.Add(10 * time.Second)},
			{Lat: 45, Lon: 6.0008, HasTime: true, Time: base.Add(20 * time.Second)},
			{Lat: 45, Lon: 6.0012},
		},
	}
	a := stats.Compute(track, config.Default()).Activity

	if !a.HasStartEnd {
		t.Fatalf("HasStartEnd = false, want true: two points carry timestamps")
	}
	if !a.Start.Equal(base.Add(10 * time.Second)) {
		t.Errorf("Start = %v, want the first timestamped point", a.Start)
	}
	if !a.End.Equal(base.Add(20 * time.Second)) {
		t.Errorf("End = %v, want the last timestamped point", a.End)
	}
}

func TestComputeActivityEmptyTrack(t *testing.T) {
	a := stats.Compute(gpx.Track{}, config.Default()).Activity
	if a.HasStartEnd || a.HasMetadataTime || a.Name != "" || a.Creator != "" || a.Type != "" {
		t.Errorf("empty track should report no identity at all: %+v", a)
	}
}

func TestElevationProfileDistanceMatchesReportedDistance(t *testing.T) {
	// If the chart's x-axis kept the phantom kilometres, the chart and the
	// statistic would disagree about how long the activity was.
	track := twoSegmentTrack()
	prof, ok := stats.ElevationOverDistance(track, 50)
	if !ok {
		t.Fatalf("expected a usable elevation profile")
	}
	want := stats.Compute(track, config.Default()).TotalDistanceKm
	if math.Abs(prof.XMax-want) > 1e-9 {
		t.Errorf("profile XMax = %v, want the reported distance %v", prof.XMax, want)
	}
}
