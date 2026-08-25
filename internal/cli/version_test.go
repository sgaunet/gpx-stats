package cli_test

import (
	"bytes"
	"encoding/json"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/cli"
)

const (
	testSHA  = "96c9f3edee5d1bb1c655910e2a42b8b0d871144c"
	testDate = "2026-07-07T18:58:31Z"
)

// stamp sets the link-time variables for one test and restores them afterwards.
// -count=2 makes the restore mandatory: the second run shares the process with
// the first, so a test that left a version behind would silently change what the
// next test observes. Never call t.Parallel in a test that uses this — the vars
// are package state and -race is on.
func stamp(t *testing.T, version, commit, date string) {
	t.Helper()
	prevV, prevC, prevD := cli.Version, cli.Commit, cli.Date
	t.Cleanup(func() { cli.Version, cli.Commit, cli.Date = prevV, prevC, prevD })
	cli.Version, cli.Commit, cli.Date = version, commit, date
}

// renderFrom is the bridge call every fallback test shares.
func renderFrom(version, commit, date string, bi *debug.BuildInfo) string {
	var b bytes.Buffer
	cli.RenderVersionFrom(&b, version, commit, date, bi)
	return b.String()
}

// rowValue returns the value a labelled row carries. hasRow answers "is this
// exact pair present"; the fallback tests need the value itself, because what
// they can assert is its shape rather than its text.
func rowValue(t *testing.T, out, label string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^ *` + regexp.QuoteMeta(label) + ` {2,}(.*)$`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no row labelled %q in:\n%s", label, out)
	}
	return m[1]
}

// buildInfo assembles the minimum BuildInfo the resolver reads.
func buildInfo(mainVersion string, settings ...debug.BuildSetting) *debug.BuildInfo {
	return &debug.BuildInfo{
		Main:     debug.Module{Version: mainVersion},
		Settings: settings,
	}
}

// TestVersionFlagPrintsStampedBuild is the only test that catches the -X symbols
// going dead: it stamps the exported vars the linker targets and asserts they
// reach the output through cli.Run. Keep it going through cli.Run.
func TestVersionFlagPrintsStampedBuild(t *testing.T) {
	stamp(t, "0.4.0", testSHA, testDate)

	code, out, errOut := run("--version")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
	if !hasRow(out, "gpx-stats", "0.4.0", "") {
		t.Errorf("missing version row:\n%s", out)
	}
	if !hasRow(out, "commit", testSHA, "") {
		t.Errorf("missing commit row:\n%s", out)
	}
	if !hasRow(out, "built", testDate, "") {
		t.Errorf("missing built row:\n%s", out)
	}
	if !hasLabel(out, "go") {
		t.Errorf("missing go row:\n%s", out)
	}
	// The SHA is reported whole; truncating would report something other than
	// what was stamped.
	if !strings.Contains(out, testSHA) {
		t.Errorf("commit was truncated:\n%s", out)
	}
}

// TestVersionNeedsNoPath pins the placement decision: the flag is honoured
// before the positional-argument check, so it needs no file.
func TestVersionNeedsNoPath(t *testing.T) {
	code, out, errOut := run("--version")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if strings.Contains(errOut, "usage:") {
		t.Errorf("printed usage instead of the version block:\n%s", errOut)
	}
	if !hasLabel(out, "commit") {
		t.Errorf("expected version block on stdout, got:\n%s", out)
	}
}

func TestVersionIgnoresPathAndJSON(t *testing.T) {
	code, out, errOut := run("--version", samplePath())
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	if !hasLabel(out, "commit") {
		t.Errorf("expected version block, got:\n%s", out)
	}
	if hasLabel(out, "Total distance") {
		t.Errorf("--version should short-circuit the report:\n%s", out)
	}

	// --version is text-only: the JSON contract covers statistics, and its
	// schema is additionalProperties:false, so a version object could not
	// validate against it.
	code, out, errOut = run("--json", "--version")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err == nil {
		t.Errorf("--version emitted JSON, want the text block:\n%s", out)
	}
}

// TestVersionAfterPathIsUsageError pins pre-existing flag behaviour: Go's flag
// package stops at the first non-flag argument, so a trailing --version is a
// positional. Shared with every other flag; pinned so nobody "fixes" it.
func TestVersionAfterPathIsUsageError(t *testing.T) {
	code, _, _ := run(samplePath(), "--version")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestVersionIgnoresNoColor(t *testing.T) {
	stamp(t, "0.4.0", testSHA, testDate)

	_, plain, _ := run("--version")
	_, noColor, _ := run("--no-color", "--version")
	if plain != noColor {
		t.Errorf("--no-color changed the block:\n%q\nvs\n%q", plain, noColor)
	}
}

// TestVersionFallbackIsPlausible exercises the real ReadBuildInfo path. It
// asserts shapes rather than literals: a developer with GOFLAGS=-buildvcs=true
// would give the test binary real vcs settings, and a literal "unknown"
// assertion would break for them.
func TestVersionFallbackIsPlausible(t *testing.T) {
	stamp(t, "", "", "")

	code, out, errOut := run("--version")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, errOut)
	}

	version := rowValue(t, out, "gpx-stats")
	if version == "" || version == "unknown" {
		t.Errorf("version = %q, want a build-info version", version)
	}

	commit := rowValue(t, out, "commit")
	if !regexp.MustCompile(`^(unknown|[0-9a-f]{40})$`).MatchString(commit) {
		t.Errorf("commit = %q, want unknown or a 40-hex SHA", commit)
	}

	built := rowValue(t, out, "built")
	if built != "unknown" {
		if _, err := time.Parse(time.RFC3339, built); err != nil {
			t.Errorf("built = %q, want unknown or RFC3339: %v", built, err)
		}
	}
}

func TestVersionLdflagsBeatBuildInfo(t *testing.T) {
	out := renderFrom("0.4.0", testSHA, testDate, buildInfo("v9.9.9",
		debug.BuildSetting{Key: "vcs.revision", Value: strings.Repeat("a", 40)},
		debug.BuildSetting{Key: "vcs.time", Value: "2020-01-01T00:00:00Z"},
	))

	if !hasRow(out, "gpx-stats", "0.4.0", "") {
		t.Errorf("build info overrode the stamped version:\n%s", out)
	}
	if !hasRow(out, "commit", testSHA, "") {
		t.Errorf("build info overrode the stamped commit:\n%s", out)
	}
	if !hasRow(out, "built", testDate, "") {
		t.Errorf("build info overrode the stamped date:\n%s", out)
	}
}

func TestVersionFallsBackToBuildInfo(t *testing.T) {
	out := renderFrom("", "", "", buildInfo("v0.4.0",
		debug.BuildSetting{Key: "vcs.revision", Value: testSHA},
		debug.BuildSetting{Key: "vcs.time", Value: testDate},
	))

	// The leading "v" is trimmed so one tag reports one string however it was
	// built: GoReleaser's {{ .Version }} carries no "v", Main.Version does.
	if !hasRow(out, "gpx-stats", "0.4.0", "") {
		t.Errorf("want version 0.4.0 with the v trimmed:\n%s", out)
	}
	if !hasRow(out, "commit", testSHA, "") {
		t.Errorf("want commit from vcs.revision:\n%s", out)
	}
	if !hasRow(out, "built", testDate, "") {
		t.Errorf("want built from vcs.time:\n%s", out)
	}
}

func TestVersionUnknownWhenNothingIsStamped(t *testing.T) {
	out := renderFrom("", "", "", nil)

	for _, label := range []string{"gpx-stats", "commit", "built"} {
		if !hasRow(out, label, "unknown", "") {
			t.Errorf("row %q = %q, want unknown:\n%s", label, rowValue(t, out, label), out)
		}
	}
	// The toolchain row never depends on build information.
	if !hasRow(out, "go", runtime.Version()+" "+runtime.GOOS+"/"+runtime.GOARCH, "") {
		t.Errorf("go row wrong:\n%s", out)
	}
}

// TestVersionPartialBuildInfo pins that the fallback is per field: an install
// from the module proxy has a version but no vcs settings, and an untagged build
// has vcs settings but only a pseudo-version.
func TestVersionPartialBuildInfo(t *testing.T) {
	out := renderFrom("", "", "", buildInfo("",
		debug.BuildSetting{Key: "vcs.revision", Value: testSHA},
	))

	if !hasRow(out, "gpx-stats", "unknown", "") {
		t.Errorf("want unknown version:\n%s", out)
	}
	if !hasRow(out, "commit", testSHA, "") {
		t.Errorf("want commit from vcs.revision:\n%s", out)
	}
	if !hasRow(out, "built", "unknown", "") {
		t.Errorf("want unknown built:\n%s", out)
	}
}

// TestVersionBlockLayout asserts the geometry the table helper produces, which
// is what a stray section() call or a num row would break.
func TestVersionBlockLayout(t *testing.T) {
	out := renderFrom("0.4.0", testSHA, testDate, nil)

	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4:\n%s", len(lines), out)
	}
	// No leading blank line: a section() would prepend one.
	if !strings.HasPrefix(out, "  gpx-stats") {
		t.Errorf("block does not start with the version row:\n%q", out)
	}

	// All four values start in the same column: 2 (indent) + 9 ("gpx-stats") + 2.
	const wantCol = 13
	for _, want := range []string{"0.4.0", testSHA, testDate} {
		for _, line := range lines {
			if idx := strings.Index(line, want); idx >= 0 && idx != wantCol {
				t.Errorf("value %q starts at column %d, want %d: %q", want, idx, wantCol, line)
			}
		}
	}

	for _, line := range lines {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("trailing whitespace: %q", line)
		}
	}
}
