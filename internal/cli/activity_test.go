package cli_test

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

func fullActivity() stats.Activity {
	start := time.Date(2023, 6, 15, 8, 0, 0, 0, time.UTC)
	return stats.Activity{
		Creator:         "Garmin Edge 530",
		Name:            "Morning Leg",
		Type:            "running",
		MetadataTime:    start.Add(-time.Minute),
		HasMetadataTime: true,
		Start:           start,
		End:             start.Add(time.Hour),
		HasStartEnd:     true,
	}
}

func TestWriteTextActivityFull(t *testing.T) {
	var buf bytes.Buffer
	cli.WriteText(&buf, stats.Result{Activity: fullActivity(), PointCount: 3})
	out := buf.String()

	// The terminal shows a human time; RFC3339 stays on the machine surfaces
	// (see TestWriteJSONActivityPopulated).
	for _, want := range []struct{ label, value string }{
		{"Activity", "Morning Leg"},
		{"Type", "running"},
		{"Recorded by", "Garmin Edge 530"},
		{"Start", "2023-06-15 08:00:00 UTC"},
		{"End", "2023-06-15 09:00:00 UTC"},
		// Never "Date": this is when the file was written, not when the
		// activity happened.
		{"File time", "2023-06-15 07:59:00 UTC"},
	} {
		if !hasRow(out, want.label, want.value, "") {
			t.Errorf("output missing row %q %q\n%s", want.label, want.value, out)
		}
	}
	if strings.Contains(out, "unavailable (no identity metadata") {
		t.Errorf("a fully identified activity should not report identity as unavailable\n%s", out)
	}
}

func TestWriteTextActivityAbsent(t *testing.T) {
	var buf bytes.Buffer
	cli.WriteText(&buf, stats.Result{PointCount: 2})
	out := buf.String()

	// One line, not six negatives.
	if !hasRow(out, "Activity", "unavailable (no identity metadata in the file)", "") {
		t.Errorf("output missing the single unavailable line\n%s", out)
	}
	// hasLabel, not strings.Contains: the labels no longer carry a trailing
	// colon, so a substring check would pass vacuously and assert nothing.
	for _, absent := range []string{"Type", "Recorded by", "Start", "File time"} {
		if hasLabel(out, absent) {
			t.Errorf("output should omit %q entirely when there is no identity\n%s", absent, out)
		}
	}
	// Never a misleading zero value.
	if strings.Contains(out, "0001-01-01") {
		t.Errorf("output leaked a zero time\n%s", out)
	}
}

func TestWriteTextActivityPartial(t *testing.T) {
	var buf bytes.Buffer
	cli.WriteText(&buf, stats.Result{
		Activity:   stats.Activity{Name: "Just A Name"},
		PointCount: 2,
	})
	out := buf.String()

	if !hasRow(out, "Activity", "Just A Name", "") {
		t.Errorf("output missing the name\n%s", out)
	}
	for _, absent := range []string{"Type", "Recorded by", "Start", "File time"} {
		if hasLabel(out, absent) {
			t.Errorf("output should omit %q when only a name is present\n%s", absent, out)
		}
	}
	if strings.Contains(out, "unavailable (no identity") {
		t.Errorf("a named activity should not report identity as unavailable\n%s", out)
	}
}

// TestWriteTextSegmentNote pins the note that explains a distance the reader
// would otherwise be unable to reconcile with a tool that joins the segments.
// It is a line of its own beneath the distance row: appended to the row it would
// wrap on any normal terminal, burying the number it qualifies.
func TestWriteTextSegmentNote(t *testing.T) {
	tests := []struct {
		name     string
		segments int
		wantNote string
		unwanted string
	}{
		{
			name:     "single segment says nothing",
			segments: 1,
			unwanted: "segments",
		},
		{
			name:     "multiple segments explain themselves",
			segments: 3,
			wantNote: "3 segments; gaps between them are not counted",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cli.WriteText(&buf, stats.Result{TotalDistanceKm: 5.5, SegmentCount: tc.segments})
			out := buf.String()
			// hasRow anchors at end of line, so this also proves the note is
			// never appended to the distance row itself.
			if !hasRow(out, "Total distance", "5.50", "km") {
				t.Errorf("output missing the distance row\n%s", out)
			}
			if tc.wantNote != "" && !regexp.MustCompile(`(?m)^ +`+regexp.QuoteMeta(tc.wantNote)+`$`).MatchString(out) {
				t.Errorf("output missing %q on its own indented line\n%s", tc.wantNote, out)
			}
			if tc.unwanted != "" && strings.Contains(out, tc.unwanted) {
				t.Errorf("output should not mention %q\n%s", tc.unwanted, out)
			}
		})
	}
}

func TestWriteJSONActivityAlwaysPresent(t *testing.T) {
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, stats.Result{PointCount: 2}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v", err)
	}

	// The object is always there; its members are null rather than "".
	act, ok := got["activity"].(map[string]any)
	if !ok {
		t.Fatalf("activity = %v, want an object even when nothing is known", got["activity"])
	}
	for _, key := range []string{"creator", "name", "type", "metadataRfc3339", "startRfc3339", "endRfc3339"} {
		v, present := act[key]
		if !present {
			t.Errorf("activity.%s is missing; every field must be present", key)
		}
		if v != nil {
			t.Errorf("activity.%s = %v, want null when absent", key, v)
		}
	}
}

func TestWriteJSONActivityPopulated(t *testing.T) {
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, stats.Result{Activity: fullActivity(), SegmentCount: 2}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	act, ok := got["activity"].(map[string]any)
	if !ok {
		t.Fatalf("activity is not an object: %v", got["activity"])
	}
	for key, want := range map[string]string{
		"creator":         "Garmin Edge 530",
		"name":            "Morning Leg",
		"type":            "running",
		"metadataRfc3339": "2023-06-15T07:59:00Z",
		"startRfc3339":    "2023-06-15T08:00:00Z",
		"endRfc3339":      "2023-06-15T09:00:00Z",
	} {
		if act[key] != want {
			t.Errorf("activity.%s = %v, want %q", key, act[key], want)
		}
	}
	if got["segmentCount"] != float64(2) {
		t.Errorf("segmentCount = %v, want 2", got["segmentCount"])
	}
}
