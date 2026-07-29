package cli_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// rowPattern matches a rendered metric row and captures its label, the run of
// spaces before the value, and the value itself. Notes sit at a four-space
// indent and use single spaces, so they never match.
var rowPattern = regexp.MustCompile(`^  (\S.*?)( {2,})(\S.*)$`)

// numericValue matches the values the layout right-aligns: plain numbers and
// durations. A timestamp starts with a digit too, which is why this is anchored
// rather than a leading-digit test.
var numericValue = regexp.MustCompile(`^(\d+(\.\d+)?|\d+h\d{2}m\d{2}s|\d+m\d{2}s|\d+s)$`)

// statsRows returns the metric rows of a report, stopping before the splits
// table — that is a grid with its own columns, not part of the label/value band.
func statsRows(out string) [][]string {
	body, _, _ := strings.Cut(out, "Kilometer splits")
	var rows [][]string
	for line := range strings.SplitSeq(body, "\n") {
		if m := rowPattern.FindStringSubmatch(line); m != nil {
			rows = append(rows, m)
		}
	}
	return rows
}

func fullResult() stats.Result {
	r := effortRateResult()
	r.TotalTime = 2 * time.Hour
	r.MovingTime = time.Hour + 50*time.Minute
	r.PauseTime = 10 * time.Minute
	r.PauseCount = 2
	r.AvgSpeedKmh = 2.75
	r.AvgMovingSpeedKmh = 3.0
	r.Splits = []stats.KmSplit{{Index: 1, DistanceKm: 1, Duration: 20 * time.Minute, SpeedKmh: 3}}
	return r
}

// TestColumnsAlign asserts the promise the layout makes: one label column and
// one value column for the whole report, whichever sections a given file fills.
// Numbers share a right edge; prose shares the left edge of the same band.
func TestColumnsAlign(t *testing.T) {
	noEle := stats.Result{TotalDistanceKm: 1.0, PointCount: 3}
	noTimes := effortResult()

	for _, tc := range []struct {
		name string
		res  stats.Result
	}{
		{"full", fullResult()},
		{"no elevation", noEle},
		{"no timestamps", noTimes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cli.WriteText(&buf, tc.res)
			rows := statsRows(buf.String())
			if len(rows) == 0 {
				t.Fatalf("no rows rendered:\n%s", buf.String())
			}

			longestLabel := 0
			for _, m := range rows {
				longestLabel = max(longestLabel, len(m[1]))
			}
			// The value field begins one gap past the widest label. Numbers are
			// right-aligned inside it, prose starts at its left edge.
			fieldStart := 2 + longestLabel + 2

			numEnd := -1
			for _, m := range rows {
				label, value := m[1], m[3]
				textStart := 2 + len(label) + len(m[2])
				first := strings.Fields(value)[0]

				if !numericValue.MatchString(first) {
					if textStart != fieldStart {
						t.Errorf("prose value starts at %d, want %d: %q", textStart, fieldStart, label)
					}
					continue
				}
				end := textStart + len(first)
				if numEnd == -1 {
					numEnd = end
				} else if end != numEnd {
					t.Errorf("numeric value for %q ends at %d, want %d — decimal points must line up",
						label, end, numEnd)
				}
				if end < fieldStart {
					t.Errorf("numeric value for %q ends at %d, before the field starts at %d",
						label, end, fieldStart)
				}
			}
		})
	}
}

// TestLabelColumnFollowsData is the point of measuring widths rather than fixing
// them: a track without elevation never renders the 36-character effort labels,
// so its whole report is narrower.
func TestLabelColumnFollowsData(t *testing.T) {
	var wide, narrow bytes.Buffer
	cli.WriteText(&wide, fullResult())
	cli.WriteText(&narrow, stats.Result{TotalDistanceKm: 1.0, PointCount: 3})

	wideAt := columnOf(t, wide.String(), "Total distance")
	narrowAt := columnOf(t, narrow.String(), "Total distance")
	if narrowAt >= wideAt {
		t.Errorf("value column is at %d without elevation and %d with it; "+
			"a sparse report must not pay for the effort labels", narrowAt, wideAt)
	}
}

// columnOf returns where the value column starts, derived from the widest label
// in the report rather than from the row asked about.
func columnOf(t *testing.T, out, label string) int {
	t.Helper()
	rows := statsRows(out)
	longest, found := 0, false
	for _, m := range rows {
		longest = max(longest, len(m[1]))
		if m[1] == label {
			found = true
		}
	}
	if !found {
		t.Fatalf("no row for %q in:\n%s", label, out)
	}
	return 2 + longest + 2
}

// TestSplitsTableWidths is the regression test for the hand-tuned "%2d … %-8s"
// format this replaces: it truncated nothing but mis-aligned every row once an
// index passed 99 or a duration passed nine characters.
func TestSplitsTableWidths(t *testing.T) {
	r := stats.Result{HasTimes: true}
	for i := 1; i <= 120; i++ {
		d := 20 * time.Minute
		if i == 7 {
			d = time.Hour + 5*time.Minute + 2*time.Second // 1h05m02s, nine characters
		}
		r.Splits = append(r.Splits, stats.KmSplit{
			Index: i, DistanceKm: 1.05, Duration: d, SpeedKmh: 3.15,
		})
	}

	var buf bytes.Buffer
	cli.RenderSplitsPlain(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "1h05m02s") {
		t.Errorf("the nine-character duration is missing:\n%s", out)
	}

	lines := gridLines(out)
	if len(lines) != 121 { // header + 120 splits
		t.Fatalf("got %d table lines, want 121:\n%s", len(lines), out)
	}
	width := len(lines[0])
	for i, line := range lines {
		if len(line) != width {
			t.Fatalf("line %d is %d columns, header is %d:\n%q\n%q", i, len(line), width, lines[0], line)
		}
	}
	// Index 120 must be intact, not clipped to a two-digit field.
	if !strings.Contains(out, "120") {
		t.Errorf("split index 120 is missing:\n%s", out)
	}
}

func gridLines(out string) []string {
	var lines []string
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(line, "  ") && strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// TestNoTrailingWhitespace is the invariant padding code violates most easily,
// and the one that makes a diff of the output noisy for no reason.
func TestNoTrailingWhitespace(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  stats.Result
	}{
		{"full", fullResult()},
		{"no elevation", stats.Result{TotalDistanceKm: 1.0, PointCount: 3}},
		{"no timestamps", effortResult()},
		{"segmented", stats.Result{TotalDistanceKm: 5.5, SegmentCount: 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cli.WriteText(&buf, tc.res)
			for i, line := range strings.Split(buf.String(), "\n") {
				if line != strings.TrimRight(line, " \t") {
					t.Errorf("line %d has trailing whitespace: %q", i+1, line)
				}
			}
		})
	}
}

// TestHumanTimestamps pins both halves of a deliberate divergence: the terminal
// reads a human time, the machine surface keeps RFC3339.
func TestHumanTimestamps(t *testing.T) {
	res := stats.Result{Activity: fullActivity(), PointCount: 3}

	var text bytes.Buffer
	cli.WriteText(&text, res)
	if !strings.Contains(text.String(), "2023-06-15 08:00:00 UTC") {
		t.Errorf("text output should carry a human timestamp:\n%s", text.String())
	}
	if strings.Contains(text.String(), "2023-06-15T08:00:00Z") {
		t.Errorf("text output should not carry RFC3339:\n%s", text.String())
	}

	var machine bytes.Buffer
	if err := cli.WriteJSON(&machine, res); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !strings.Contains(machine.String(), "2023-06-15T08:00:00Z") {
		t.Errorf("JSON must keep RFC3339:\n%s", machine.String())
	}
}
