package gpx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
