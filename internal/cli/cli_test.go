package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/cli"
)

func samplePath() string { return filepath.Join("..", "..", "testdata", "sample.gpx") }

func run(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := cli.Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRunStatsText(t *testing.T) {
	code, out, errOut := run(samplePath())
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !hasLabel(out, "Total distance") {
		t.Errorf("expected stats text, got:\n%s", out)
	}
}

func TestRunStatsJSON(t *testing.T) {
	code, out, errOut := run("--json", samplePath())
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if _, ok := m["totalDistanceKm"]; !ok {
		t.Errorf("json missing totalDistanceKm")
	}
}

func TestRunMissingFile(t *testing.T) {
	code, _, errOut := run(filepath.Join("..", "..", "testdata", "does-not-exist.gpx"))
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(errOut, "cannot open") {
		t.Errorf("expected actionable error, got: %s", errOut)
	}
}

func TestRunNoPath(t *testing.T) {
	code, _, _ := run()
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage)", code)
	}
}

func TestRunUnknownFlag(t *testing.T) {
	code, _, _ := run("--nope", samplePath())
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage)", code)
	}
}

func TestRunPauseFlagsParsed(t *testing.T) {
	code, out, errOut := run("--pause-speed", "2.0", "--pause-duration", "20s", samplePath())
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !hasLabel(out, "Pause time") {
		t.Errorf("expected pause stats, got:\n%s", out)
	}
}

// TestRunElevationNoiseFlagParsed checks the threshold is reachable from the
// command line and actually changes the reported ascent: a huge threshold
// filters every rise away, so gain collapses to zero.
func TestRunElevationNoiseFlagParsed(t *testing.T) {
	code, out, errOut := run("--elevation-noise", "0", samplePath())
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !hasLabel(out, "Ascending elev.") {
		t.Fatalf("expected elevation stats, got:\n%s", out)
	}

	code, filtered, errOut := run("--elevation-noise", "1000", samplePath())
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !hasRow(filtered, "Ascending elev.", "0", "m") {
		t.Errorf("a 1000 m threshold should filter all gain, got:\n%s", filtered)
	}
	if out == filtered {
		t.Errorf("elevation-noise had no effect on the output")
	}
}

func TestRunInvalidElevationNoise(t *testing.T) {
	code, _, errOut := run("--elevation-noise", "-1", samplePath())
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(errOut, "invalid configuration") {
		t.Errorf("expected configuration error, got: %s", errOut)
	}
}

func TestRunInvalidPauseDuration(t *testing.T) {
	// pause-duration of 0 fails config validation → usage exit code.
	code, _, errOut := run("--pause-duration", "0s", samplePath())
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, errOut)
	}
	if !strings.Contains(errOut, "invalid configuration") {
		t.Errorf("expected configuration error, got: %s", errOut)
	}
}

func TestRunNoTimeFixture(t *testing.T) {
	code, out, _ := run(filepath.Join("..", "..", "testdata", "no_time.gpx"))
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "unavailable") {
		t.Errorf("expected unavailable notice for timeless track:\n%s", out)
	}
}
