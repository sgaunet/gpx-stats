package cli_test

import (
	"bytes"
	"encoding/json"
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

	for _, want := range []string{
		"Activity:          Morning Leg",
		"Type:              running",
		"Recorded by:       Garmin Edge 530",
		"Start:             2023-06-15T08:00:00Z",
		"End:               2023-06-15T09:00:00Z",
		// Never "Date": this is when the file was written, not when the
		// activity happened.
		"File time:         2023-06-15T07:59:00Z",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
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
	if !strings.Contains(out, "Activity:          unavailable (no identity metadata in the file)") {
		t.Errorf("output missing the single unavailable line\n%s", out)
	}
	for _, absent := range []string{"Type:", "Recorded by:", "Start:", "File time:"} {
		if strings.Contains(out, absent) {
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

	if !strings.Contains(out, "Activity:          Just A Name") {
		t.Errorf("output missing the name\n%s", out)
	}
	for _, absent := range []string{"Type:", "Recorded by:", "Start:", "File time:", "unavailable (no identity"} {
		if strings.Contains(out, absent) {
			t.Errorf("output should omit %q when only a name is present\n%s", absent, out)
		}
	}
}

func TestWriteTextSegmentNote(t *testing.T) {
	tests := []struct {
		name     string
		segments int
		want     string
		unwanted string
	}{
		{
			name:     "single segment says nothing",
			segments: 1,
			want:     "Total distance:    5.50 km\n",
			unwanted: "segments",
		},
		{
			name:     "multiple segments explain themselves",
			segments: 3,
			want:     "Total distance:    5.50 km (3 segments; gaps between them are not counted)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cli.WriteText(&buf, stats.Result{TotalDistanceKm: 5.5, SegmentCount: tc.segments})
			out := buf.String()
			if !strings.Contains(out, tc.want) {
				t.Errorf("output missing %q\n%s", tc.want, out)
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
