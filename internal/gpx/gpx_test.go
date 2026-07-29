package gpx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/gpx"
)

const (
	bigLimit   = int64(10 << 20)
	manyPoints = 1_000_000
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

func parseFixture(t *testing.T, name string) (gpx.Track, error) {
	t.Helper()
	return gpx.Parse(strings.NewReader(string(fixture(t, name))), bigLimit, manyPoints)
}

func TestParseValid(t *testing.T) {
	track, err := parseFixture(t, "sample.gpx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(track.Points) != 8 {
		t.Fatalf("got %d points, want 8", len(track.Points))
	}
	if !track.HasElevation {
		t.Errorf("expected HasElevation true")
	}
	if !track.HasTimes {
		t.Errorf("expected HasTimes true")
	}
	p := track.Points[0]
	if p.Lat != 45.0 || p.Lon != 6.0 || !p.HasEle || p.Ele != 100.0 || !p.HasTime {
		t.Errorf("first point decoded incorrectly: %+v", p)
	}
}

func TestParseMissingElevation(t *testing.T) {
	track, err := parseFixture(t, "no_ele.gpx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if track.HasElevation {
		t.Errorf("expected HasElevation false")
	}
	if !track.HasTimes {
		t.Errorf("expected HasTimes true")
	}
}

func TestParseMissingTime(t *testing.T) {
	track, err := parseFixture(t, "no_time.gpx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !track.HasElevation {
		t.Errorf("expected HasElevation true")
	}
	if track.HasTimes {
		t.Errorf("expected HasTimes false")
	}
}

func TestParseSinglePoint(t *testing.T) {
	track, err := parseFixture(t, "single_point.gpx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(track.Points) != 1 {
		t.Fatalf("got %d points, want 1", len(track.Points))
	}
	if track.HasElevation || track.HasTimes {
		t.Errorf("single point should leave HasElevation/HasTimes false")
	}
}

// TestParseMaliciousRejected is the security-critical case: a GPX with a DTD
// declaring entities (an XXE external entity and an expansion bomb) must be
// rejected safely without a crash, file read, or network access.
func TestParseMaliciousRejected(t *testing.T) {
	_, err := parseFixture(t, "malicious.gpx")
	if err == nil {
		t.Fatalf("expected malicious GPX to be rejected, got nil error")
	}
	// The rejection must NOT contain the contents of a local file (no XXE leak).
	if strings.Contains(err.Error(), "root:") {
		t.Fatalf("error appears to contain file contents (possible XXE leak): %v", err)
	}
}

func TestParseNotGPX(t *testing.T) {
	_, err := gpx.Parse(strings.NewReader(`<html><body>nope</body></html>`), bigLimit, manyPoints)
	if err == nil {
		t.Fatalf("expected error for non-GPX input")
	}
}

func TestParseMalformedXML(t *testing.T) {
	_, err := gpx.Parse(strings.NewReader(`<gpx><trk><trkseg>`), bigLimit, manyPoints)
	if err == nil {
		t.Fatalf("expected error for truncated XML")
	}
}

func TestParseSizeLimit(t *testing.T) {
	data := fixture(t, "sample.gpx")
	_, err := gpx.Parse(strings.NewReader(string(data)), 10 /* bytes */, manyPoints)
	if err == nil {
		t.Fatalf("expected size-limit error")
	}
	if !strings.Contains(err.Error(), "maximum size") {
		t.Errorf("expected size-limit message, got: %v", err)
	}
}

func TestParsePointLimit(t *testing.T) {
	data := fixture(t, "sample.gpx")
	_, err := gpx.Parse(strings.NewReader(string(data)), bigLimit, 3 /* max points */)
	if err == nil {
		t.Fatalf("expected point-limit error")
	}
	if !strings.Contains(err.Error(), "track points") {
		t.Errorf("expected point-limit message, got: %v", err)
	}
}

func TestParseInvalidLatitude(t *testing.T) {
	bad := `<gpx version="1.1"><trk><trkseg>` +
		`<trkpt lat="200.0" lon="6.0"><ele>1</ele></trkpt>` +
		`</trkseg></trk></gpx>`
	_, err := gpx.Parse(strings.NewReader(bad), bigLimit, manyPoints)
	if err == nil {
		t.Fatalf("expected error for out-of-range latitude")
	}
}

// segmentIndices reads the segment index off every point, so a whole document's
// segmentation can be asserted as one value.
func segmentIndices(track gpx.Track) []int {
	got := make([]int, 0, len(track.Points))
	for _, p := range track.Points {
		got = append(got, p.SegmentIndex)
	}
	return got
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseSegmentIndices(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    []int
	}{
		// Every fixture that predates segment awareness must still parse to a
		// single segment, or some existing statistic would silently move.
		{"single segment", "sample.gpx", []int{0, 0, 0, 0, 0, 0, 0, 0}},
		{"no elevation", "no_ele.gpx", []int{0, 0, 0, 0}},
		{"no time", "no_time.gpx", []int{0, 0, 0, 0}},
		{"dateline", "dateline.gpx", []int{0, 0, 0, 0}},
		{"single point", "single_point.gpx", []int{0}},

		{"two segments in one track", "two_segments.gpx", []int{0, 0, 0, 1, 1, 1}},
		{"two tracks", "two_tracks.gpx", []int{0, 0, 0, 1, 1, 1}},
		// The empty <trkseg> in the middle must not burn an index: the indices
		// stay dense so they keep meaning "which run of points".
		{"empty segment between two", "empty_segment.gpx", []int{0, 0, 1, 1}},
		// A <trkpt> outside any <trkseg> is not valid GPX, but must not crash or
		// misnumber the real segment that follows it.
		{"stray point outside a segment", "stray_trkpt.gpx", []int{0, 1, 1}},
		{"bare file", "bare.gpx", []int{0, 0}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			track, err := parseFixture(t, tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := segmentIndices(track); !equalInts(got, tc.want) {
				t.Errorf("segment indices = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSameSegment(t *testing.T) {
	a := gpx.TrackPoint{SegmentIndex: 0}
	b := gpx.TrackPoint{SegmentIndex: 0}
	c := gpx.TrackPoint{SegmentIndex: 1}

	if !gpx.SameSegment(a, b) {
		t.Errorf("points in segment 0 should be in the same segment")
	}
	if gpx.SameSegment(b, c) {
		t.Errorf("points in segments 0 and 1 should not be in the same segment")
	}
	// The zero value has to mean "one segment", or every hand-built track in the
	// test suite would silently acquire boundaries.
	if !gpx.SameSegment(gpx.TrackPoint{}, gpx.TrackPoint{}) {
		t.Errorf("zero-valued points should be in the same segment")
	}
}

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		name            string
		fixture         string
		wantCreator     string
		wantName        string
		wantType        string
		wantMetadata    bool
		wantMetadataRFC string
	}{
		{
			name:            "all identity present, first track wins",
			fixture:         "two_tracks.gpx",
			wantCreator:     "gpx-stats-test suite 2",
			wantName:        "Morning Leg",
			wantType:        "running",
			wantMetadata:    true,
			wantMetadataRFC: "2023-06-15T07:59:00Z",
		},
		{
			name:        "creator and track name only",
			fixture:     "sample.gpx",
			wantCreator: "gpx-stats-test",
			wantName:    "Sample Track",
		},
		{
			// The <metadata><name> here must NOT be picked up as the activity
			// name — only the <trk><name> counts.
			fixture:     "no_coords.gpx",
			name:        "metadata name is not the track name",
			wantCreator: "gpx-stats-test",
			wantName:    "Empty Segment",
		},
		{
			name:    "nothing at all",
			fixture: "bare.gpx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			track, err := parseFixture(t, tc.fixture)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if track.Creator != tc.wantCreator {
				t.Errorf("Creator = %q, want %q", track.Creator, tc.wantCreator)
			}
			if track.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", track.Name, tc.wantName)
			}
			if track.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", track.Type, tc.wantType)
			}
			if track.HasMetadataTime != tc.wantMetadata {
				t.Fatalf("HasMetadataTime = %v, want %v", track.HasMetadataTime, tc.wantMetadata)
			}
			if tc.wantMetadata {
				if got := track.MetadataTime.Format(time.RFC3339); got != tc.wantMetadataRFC {
					t.Errorf("MetadataTime = %s, want %s", got, tc.wantMetadataRFC)
				}
			}
		})
	}
}

func TestParseIdentityMetadataTimeIsNotTheStart(t *testing.T) {
	// <metadata><time> is when the file was written, which tools routinely set
	// to the export time. Conflating it with the activity start would silently
	// misdate every analysis.
	track, err := parseFixture(t, "two_tracks.gpx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !track.MetadataTime.Before(track.Points[0].Time) {
		t.Errorf("fixture no longer distinguishes file time %s from first point %s",
			track.MetadataTime, track.Points[0].Time)
	}
}

func TestParseIdentityEdgeCases(t *testing.T) {
	longName := strings.Repeat("x", 10_000)

	tests := []struct {
		name     string
		doc      string
		wantName string
		wantType string
	}{
		{
			name:     "whitespace-only name is absent",
			doc:      `<gpx><trk><name>   </name><type>running</type></trk></gpx>`,
			wantName: "",
			wantType: "running",
		},
		{
			name:     "surrounding whitespace is trimmed",
			doc:      "<gpx><trk><name>\n  Morning Run\n  </name></trk></gpx>",
			wantName: "Morning Run",
		},
		{
			name:     "empty first name does not block a later one",
			doc:      `<gpx><trk><name></name></trk><trk><name>Second</name></trk></gpx>`,
			wantName: "Second",
		},
		{
			name:     "an over-long name is truncated, not rejected",
			doc:      `<gpx><trk><name>` + longName + `</name></trk></gpx>`,
			wantName: strings.Repeat("x", 256),
		},
		{
			name:     "waypoint name is not the track name",
			doc:      `<gpx><wpt lat="45" lon="6"><name>Summit</name></wpt><trk><name>Real</name></trk></gpx>`,
			wantName: "Real",
		},
		{
			name:     "route name is not the track name",
			doc:      `<gpx><rte><name>Planned</name></rte><trk><name>Recorded</name></trk></gpx>`,
			wantName: "Recorded",
		},
		{
			name:     "self-closing name element",
			doc:      `<gpx><trk><name/><type>cycling</type></trk></gpx>`,
			wantName: "",
			wantType: "cycling",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			track, err := gpx.Parse(strings.NewReader(tc.doc), bigLimit, manyPoints)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if track.Name != tc.wantName {
				t.Errorf("Name = %q (len %d), want %q (len %d)",
					track.Name, len(track.Name), tc.wantName, len(tc.wantName))
			}
			if track.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", track.Type, tc.wantType)
			}
		})
	}
}

func TestParseIdentityMalformedMetadataTimeIsIgnored(t *testing.T) {
	// A descriptive field must not be able to cost the user the analysis.
	doc := `<gpx><metadata><time>not-a-timestamp</time></metadata>` +
		`<trk><trkseg><trkpt lat="45" lon="6"/><trkpt lat="45" lon="6.001"/></trkseg></trk></gpx>`
	track, err := gpx.Parse(strings.NewReader(doc), bigLimit, manyPoints)
	if err != nil {
		t.Fatalf("a malformed metadata time must not fail the parse: %v", err)
	}
	if track.HasMetadataTime {
		t.Errorf("HasMetadataTime = true, want false for an unparseable value")
	}
	if len(track.Points) != 2 {
		t.Errorf("got %d points, want 2 — the track must still be read", len(track.Points))
	}
}

func TestParseIdentityDoesNotLeakHostileContent(t *testing.T) {
	// The entity bomb in malicious.gpx sits inside <trk><name>, which identity
	// parsing now decodes. The rejection must stay clean: naming the offending
	// entity is useful diagnostics, but no entity may have been *resolved*.
	_, err := parseFixture(t, "malicious.gpx")
	if err == nil {
		t.Fatalf("expected the malicious fixture to be rejected")
	}
	msg := err.Error()

	// Resolved external content. "root:" is the giveaway for /etc/passwd.
	if strings.Contains(msg, "root:") {
		t.Errorf("error message leaks external entity content: %v", err)
	}
	// An expanded bomb: &lol3; resolves to 1000 repetitions of "lol". Naming it
	// once is fine; expanding it is not.
	if strings.Count(msg, "lol") > 1 {
		t.Errorf("error message contains an expanded entity: %v", err)
	}
	// Whatever the document contains, the error stays a message, not a payload.
	if len(msg) > 500 {
		t.Errorf("error message is %d bytes; a document must not be able to inflate it", len(msg))
	}
}

func TestParseLeadingSegmentDoesNotShiftFirstPoint(t *testing.T) {
	// The opening <trkseg> arms a boundary that no earlier point can spend, so
	// the first real point must still land in segment 0.
	track, err := parseFixture(t, "sample.gpx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if track.Points[0].SegmentIndex != 0 {
		t.Errorf("first point is in segment %d, want 0", track.Points[0].SegmentIndex)
	}
}
