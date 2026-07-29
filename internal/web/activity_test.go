package web_test

import (
	"regexp"
	"strings"
	"testing"
)

func TestResultsPageShowsActivityIdentity(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "two_tracks.gpx"), nil))
	body := rec.Body.String()

	for _, want := range []string{
		"Morning Leg",
		"running",
		"gpx-stats-test suite 2",
		"2023-06-15T08:00:00Z",
		"2023-06-15T08:10:40Z",
		"2023-06-15T07:59:00Z",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("results page missing activity identity %q", want)
		}
	}
	// The second track must not rename the activity.
	if strings.Contains(body, "Evening Leg") {
		t.Errorf("results page shows the second track's name; the first should win")
	}
	// The heading names the page, not the file.
	if !strings.Contains(body, "Activity statistics") {
		t.Errorf("results page lost its heading")
	}
}

func TestResultsPageOmitsIdentityBlockWhenAbsent(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "bare.gpx"), nil))
	body := rec.Body.String()

	if !strings.Contains(body, "Activity statistics") {
		t.Fatalf("results page should still render for a file with no identity")
	}
	// A zero time must never leak into the page as though it were real.
	if strings.Contains(body, "0001-01-01") {
		t.Errorf("results page rendered a zero time")
	}
	if strings.Contains(body, "recorded by") {
		t.Errorf("results page rendered an empty creator line")
	}
}

func TestResultsPageShowsSegmentNote(t *testing.T) {
	multi := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "two_segments.gpx"), nil)).Body.String()
	if !strings.Contains(multi, "gaps between them are not counted") {
		t.Errorf("multi-segment results page should explain the reduced distance")
	}
	if !strings.Contains(multi, "2 segments") {
		t.Errorf("multi-segment results page should state the segment count")
	}

	single := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil)).Body.String()
	if strings.Contains(single, "gaps between them are not counted") {
		t.Errorf("single-segment results page should not mention segments at all")
	}
}

// hostileIdentity is a GPX document whose identity strings try to break out of
// the HTML. It lives here rather than in testdata/ so the payload is visible at
// the assertion that depends on it.
const hostileIdentity = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="&lt;script&gt;alert('creator')&lt;/script&gt;">
  <trk>
    <name>&lt;/script&gt;&lt;img src=x onerror=alert('name')&gt;</name>
    <type>&lt;b&gt;bold&lt;/b&gt;</type>
    <trkseg>
      <trkpt lat="45.0000" lon="6.0000"><ele>100</ele><time>2023-06-15T08:00:00Z</time></trkpt>
      <trkpt lat="45.0004" lon="6.0004"><ele>105</ele><time>2023-06-15T08:00:10Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>`

func TestResultsPageEscapesHostileIdentity(t *testing.T) {
	// Identity strings are the first user-controlled text ever rendered on this
	// page — everything before this feature was a number.
	rec := serve(t, testServer(t, nil), uploadRequest(t, []byte(hostileIdentity), nil))
	body := rec.Body.String()

	for _, raw := range []string{
		"<script>alert('creator')</script>",
		"<img src=x onerror=alert('name')>",
		"<b>bold</b>",
	} {
		if strings.Contains(body, raw) {
			t.Errorf("results page contains unescaped identity %q", raw)
		}
	}
	// It must still be displayed — escaped, not dropped.
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("results page should show the creator escaped, not omit it")
	}

	// The map payloads must stay numeric: identity may never leak into the
	// template.JS blocks, whose safety rests on that alphabet.
	numeric := regexp.MustCompile(`^[-0-9.,eE+\[\]]*$`)
	if p := scriptPayload(t, body, "gpx-route"); !numeric.MatchString(p) {
		t.Errorf("route payload is no longer numeric-only: %q", p)
	}
	if p := scriptPayload(t, body, "gpx-map-config"); strings.Contains(p, "alert(") {
		t.Errorf("identity leaked into the map config: %q", p)
	}
}

func TestResultsPageTruncatesOverLongName(t *testing.T) {
	doc := `<gpx version="1.1"><trk><name>` + strings.Repeat("A", 10_000) + `</name><trkseg>` +
		`<trkpt lat="45.0" lon="6.0"><ele>100</ele></trkpt>` +
		`<trkpt lat="45.001" lon="6.001"><ele>105</ele></trkpt>` +
		`</trkseg></trk></gpx>`
	body := serve(t, testServer(t, nil), uploadRequest(t, []byte(doc), nil)).Body.String()

	if strings.Contains(body, strings.Repeat("A", 257)) {
		t.Errorf("results page rendered an unbounded activity name")
	}
	if !strings.Contains(body, strings.Repeat("A", 256)) {
		t.Errorf("results page should still show the truncated name")
	}
}
