package cli_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// charDevice returns a writer that is a character device, which is the only
// thing isTerminal accepts. /dev/null is one on every platform the binary is
// built for, so the true branch is reachable without a pty and without a
// dependency.
func charDevice(t *testing.T) *os.File {
	t.Helper()
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// TestColorEnabledPrecedence pins the order in which the four inputs are
// consulted. Every case clears NO_COLOR and TERM first: a developer running the
// suite with NO_COLOR exported would otherwise get a green but meaningless run.
func TestColorEnabledPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		noColor bool
		env     map[string]string
		tty     bool
		want    bool
	}{
		{name: "terminal with a capable TERM", env: map[string]string{"TERM": "xterm-256color"}, tty: true, want: true},
		{name: "terminal with TERM unset", tty: true, want: true},
		{name: "buffer is never a terminal", env: map[string]string{"TERM": "xterm-256color"}, want: false},
		{name: "--no-color beats everything", noColor: true, env: map[string]string{"TERM": "xterm-256color"}, tty: true, want: false},
		{name: "NO_COLOR disables", env: map[string]string{"NO_COLOR": "1", "TERM": "xterm-256color"}, tty: true, want: false},
		// Per https://no-color.org the variable must be present AND non-empty.
		{name: "empty NO_COLOR does not disable", env: map[string]string{"NO_COLOR": "", "TERM": "xterm-256color"}, tty: true, want: true},
		{name: "TERM=dumb disables", env: map[string]string{"TERM": "dumb"}, tty: true, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", "")
			t.Setenv("TERM", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			var w io.Writer = &bytes.Buffer{}
			if tc.tty {
				w = charDevice(t)
			}

			if got := cli.ColorEnabled(w, tc.noColor); got != tc.want {
				t.Errorf("ColorEnabled = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	if cli.IsTerminal(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer must not be treated as a terminal")
	}

	regular, err := os.Create(filepath.Join(t.TempDir(), "out.txt"))
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer func() { _ = regular.Close() }()
	if cli.IsTerminal(regular) {
		t.Error("a regular file must not be treated as a terminal")
	}

	if !cli.IsTerminal(charDevice(t)) {
		t.Errorf("%s is a character device and must be treated as a terminal", os.DevNull)
	}
}

// TestStyleCodes pins the "bold and dim only" decision. A hue that reads well on
// a dark background is often invisible on a light one, and the renderer cannot
// know which it faces.
func TestStyleCodes(t *testing.T) {
	if got, want := cli.StyledBold("x"), "\x1b[1mx\x1b[0m"; got != want {
		t.Errorf("bold = %q, want %q", got, want)
	}
	if got, want := cli.StyledDim("x"), "\x1b[2mx\x1b[0m"; got != want {
		t.Errorf("dim = %q, want %q", got, want)
	}
	// An empty string gets no styling, so a line never carries a bare reset.
	if got := cli.StyledBold(""); got != "" {
		t.Errorf("bold(\"\") = %q, want empty", got)
	}
}

// TestNoEscapesOnBuffer protects the property that makes every other test in
// this package readable: styling is decided by the writer, so a buffer never
// sees an escape sequence and no test has to ask for plain text. A refactor that
// checked TERM before the writer would break it silently.
func TestNoEscapesOnBuffer(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	r := stats.Result{
		TotalDistanceKm: 5.5, AscendingElevationM: 120, HasElevation: true,
		TotalTime: time.Hour, MovingTime: time.Hour, HasTimes: true,
		PointCount: 100, SegmentCount: 2,
		Splits: []stats.KmSplit{{Index: 1, DistanceKm: 1, Duration: time.Hour, SpeedKmh: 3}},
	}

	var text bytes.Buffer
	cli.WriteText(&text, r)
	if strings.Contains(text.String(), "\x1b") {
		t.Errorf("WriteText leaked an escape sequence to a buffer:\n%q", text.String())
	}

	var charts bytes.Buffer
	cli.WriteCharts(&charts, gpx.Track{}, r)
	if strings.Contains(charts.String(), "\x1b") {
		t.Errorf("WriteCharts leaked an escape sequence to a buffer:\n%q", charts.String())
	}
}
