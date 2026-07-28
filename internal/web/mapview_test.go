package web_test

import (
	"encoding/json"
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/geo"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/web"
)

// mapConfigOf extracts and decodes the #gpx-map-config payload from a rendered
// page, so tests assert on what the browser actually receives.
func mapConfigOf(t *testing.T, body string) map[string]any {
	t.Helper()
	raw := scriptPayload(t, body, "gpx-map-config")
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("decoding map config %q: %v", raw, err)
	}
	return cfg
}

// scriptPayload returns the text content of <script ... id="ID">...</script>.
func scriptPayload(t *testing.T, body, id string) string {
	t.Helper()
	_, rest, found := strings.Cut(body, `id="`+id+`">`)
	if !found {
		t.Fatalf("page does not contain a script block with id %q", id)
	}
	payload, _, closed := strings.Cut(rest, "</script>")
	if !closed {
		t.Fatalf("unterminated script block for id %q", id)
	}
	return payload
}

func hasScript(body, id string) bool {
	return strings.Contains(body, `id="`+id+`">`)
}

func trackOf(latlons ...float64) gpx.Track {
	var tr gpx.Track
	for i := 0; i < len(latlons); i += 2 {
		tr.Points = append(tr.Points, gpx.TrackPoint{Lat: latlons[i], Lon: latlons[i+1]})
	}
	return tr
}

// --------------------------------------------------------------- invariants

// numericOnly is the character set the route payload may contain. Because no
// string over this alphabet can contain "</script>", injecting it as
// template.JS cannot break out of the script block.
var numericOnly = regexp.MustCompile(`^[-0-9.,eE+\[\]]*$`)

// TestRouteJSONIsNumericOnly enforces the invariant that makes the template.JS
// injection safe. It is asserted rather than assumed, because the safety of
// bypassing contextual escaping rests entirely on it.
func TestRouteJSONIsNumericOnly(t *testing.T) {
	cases := []struct {
		name  string
		track gpx.Track
	}{
		{"extremes", trackOf(90, 180, -90, -180)},
		{"negatives", trackOf(-45.123456, -6.654321, -45.2, -6.7)},
		{"high precision", trackOf(45.123456789012345, 6.987654321098765, 45.2, 6.9)},
		{"near zero", trackOf(0.000001, -0.000001, 0.000002, -0.000002)},
		{"antimeridian", trackOf(-16.5, 179.99999, -16.5, -179.99999)},
		{"tiny deltas", trackOf(45.000001, 6.000001, 45.000002, 6.000002)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := web.BuildMapView(geo.BuildRoute(tc.track), false)
			if err != nil {
				t.Fatalf("BuildMapView: %v", err)
			}
			payload := string(v.RouteJSON)
			if !numericOnly.MatchString(payload) {
				t.Errorf("route payload contains unexpected characters: %q", payload)
			}
			if strings.Contains(strings.ToLower(payload), "</script") {
				t.Errorf("route payload could break out of the script block: %q", payload)
			}
		})
	}
}

func TestBuildMapViewEmptyRouteHasNoCoordinates(t *testing.T) {
	v, err := web.BuildMapView(geo.Route{}, true)
	if err != nil {
		t.Fatalf("BuildMapView: %v", err)
	}
	if v.HasRoute {
		t.Errorf("HasRoute = true, want false for an empty route")
	}
	if v.RouteJSON != "" {
		t.Errorf("RouteJSON = %q, want empty", v.RouteJSON)
	}
}

// ------------------------------------------------------------- results page

func TestResultsPageEmbedsRoute(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil))
	body := rec.Body.String()

	cfg := mapConfigOf(t, body)
	if cfg["hasRoute"] != true {
		t.Errorf("hasRoute = %v, want true", cfg["hasRoute"])
	}
	if cfg["dropzone"] != false {
		t.Errorf("dropzone = %v, want false on the results page", cfg["dropzone"])
	}

	// The embedded coordinates must match the parsed track exactly: same
	// count, same order, nothing dropped (FR-001a, SC-005a).
	want := geo.BuildRoute(mustParse(t, fixtureBytes(t, "sample.gpx")))
	var got []float64
	if err := json.Unmarshal([]byte(scriptPayload(t, body, "gpx-route")), &got); err != nil {
		t.Fatalf("decoding route payload: %v", err)
	}
	if len(got) != len(want.Coords) {
		t.Fatalf("route payload has %d values, want %d", len(got), len(want.Coords))
	}
	for i := range got {
		if math.Abs(got[i]-want.Coords[i]) > 1e-9 {
			t.Errorf("coordinate %d = %v, want %v", i, got[i], want.Coords[i])
		}
	}
	if pc, ok := cfg["pointCount"].(float64); !ok || int(pc) != want.PointCount {
		t.Errorf("pointCount = %v, want %d", cfg["pointCount"], want.PointCount)
	}
}

func TestResultsPageSinglePoint(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "single_point.gpx"), nil))
	cfg := mapConfigOf(t, rec.Body.String())

	if pc, ok := cfg["pointCount"].(float64); !ok || int(pc) != 1 {
		t.Errorf("pointCount = %v, want 1", cfg["pointCount"])
	}
	if cfg["useBounds"] != false {
		t.Errorf("useBounds = %v, want false: fitBounds on one point over-zooms", cfg["useBounds"])
	}
	if _, present := cfg["bounds"]; present {
		t.Errorf("bounds should be omitted when useBounds is false")
	}
	if z, ok := cfg["zoom"].(float64); !ok || int(z) != geo.DefaultPointZoom {
		t.Errorf("zoom = %v, want %d", cfg["zoom"], geo.DefaultPointZoom)
	}
}

func TestResultsPageNoCoordinatesOmitsMap(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "no_coords.gpx"), nil))
	body := rec.Body.String()

	if hasScript(body, "gpx-map-config") {
		t.Errorf("a track with no coordinates must not render a map section")
	}
	if hasScript(body, "gpx-route") {
		t.Errorf("a track with no coordinates must not embed a route payload")
	}
	if !strings.Contains(body, "Activity statistics") {
		t.Errorf("the statistics page should still render")
	}
}

func TestResultsPageDatelineDoesNotWrapTheWorld(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "dateline.gpx"), nil))
	cfg := mapConfigOf(t, rec.Body.String())

	bounds, ok := cfg["bounds"].([]any)
	if !ok || len(bounds) != 4 {
		t.Fatalf("bounds = %v, want a 4-element array", cfg["bounds"])
	}
	minLon, _ := bounds[1].(float64)
	maxLon, _ := bounds[3].(float64)
	if span := maxLon - minLon; span > 1 {
		t.Errorf("longitude span = %v, want ~0.2: the antimeridian was not unwrapped", span)
	}
}

// TestResultsStatisticsUnchanged is the regression guard: adding a map must not
// disturb anything already on the page.
func TestResultsStatisticsUnchanged(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil))
	body := rec.Body.String()

	for _, want := range []string{
		"Activity statistics",
		"Distance",
		"Ascending elevation",
		"Total time",
		"Moving time",
		"Pause time",
		"Avg speed",
		"Kilometer splits",
		"<svg",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("results page is missing %q", want)
		}
	}
}

// ------------------------------------------------------------- landing page

func TestIndexPageMapConfig(t *testing.T) {
	rec := serve(t, testServer(t, nil), getRequest(t, "/"))
	body := rec.Body.String()

	cfg := mapConfigOf(t, body)
	if cfg["hasRoute"] != false {
		t.Errorf("hasRoute = %v, want false on the landing page", cfg["hasRoute"])
	}
	if cfg["dropzone"] != true {
		t.Errorf("dropzone = %v, want true on the landing page", cfg["dropzone"])
	}
	if hasScript(body, "gpx-route") {
		t.Errorf("the landing page must not embed a route payload")
	}
	// A fixed default view: no geolocation, no inference from a prior upload.
	center, ok := cfg["center"].([]any)
	if !ok || len(center) != 2 {
		t.Fatalf("center = %v, want a 2-element array", cfg["center"])
	}
}

// TestIndexFormUnchanged pins the upload form's contract. The drop path works
// by submitting this exact form, so a rename here silently breaks both paths.
func TestIndexFormUnchanged(t *testing.T) {
	body := serve(t, testServer(t, nil), getRequest(t, "/")).Body.String()

	for _, want := range []string{
		`action="/analyze"`,
		`method="post"`,
		`enctype="multipart/form-data"`,
		`name="file"`,
		`id="file"`,
		`name="pause_speed"`,
		`name="pause_duration"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("upload form is missing %q", want)
		}
	}
}

func TestRenderedLayersMatchTable(t *testing.T) {
	cfg := mapConfigOf(t, serve(t, testServer(t, nil), getRequest(t, "/")).Body.String())

	layers, ok := cfg["layers"].([]any)
	if !ok {
		t.Fatalf("layers = %v, want an array", cfg["layers"])
	}
	if len(layers) != len(web.BaseLayers) {
		t.Fatalf("rendered %d layers, want %d", len(layers), len(web.BaseLayers))
	}
	if len(layers) < 3 {
		t.Errorf("only %d layers offered, want at least 3", len(layers))
	}

	var defaults int
	for i, raw := range layers {
		l, lok := raw.(map[string]any)
		if !lok {
			t.Fatalf("layer %d is not an object", i)
		}
		want := web.BaseLayers[i]
		if l["key"] != want.Key {
			t.Errorf("layer %d key = %v, want %q", i, l["key"], want.Key)
		}
		if l["url"] != want.URLTemplate {
			t.Errorf("layer %d url = %v, want %q", i, l["url"], want.URLTemplate)
		}
		if l["attribution"] != want.Attribution {
			t.Errorf("layer %d attribution does not match the table", i)
		}
		if l["default"] == true {
			defaults++
		}
	}
	if defaults != 1 {
		t.Errorf("%d layers marked default in the rendered config, want exactly 1", defaults)
	}
}

// ---------------------------------------------------------- privacy & scope

// TestPrivacyNoticeIsAccurate guards the claim the pages make. Once a map
// loads, "no data leaves this server" is no longer true: the browser talks to
// the tile provider. The notice must say so.
func TestPrivacyNoticeIsAccurate(t *testing.T) {
	srv := testServer(t, nil)
	pages := map[string]string{
		"index":   serve(t, srv, getRequest(t, "/")).Body.String(),
		"results": serve(t, srv, uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil)).Body.String(),
	}
	for name, body := range pages {
		t.Run(name, func(t *testing.T) {
			if strings.Contains(body, "no data leaves this server") {
				t.Errorf("page still claims that no data leaves the server, which map tiles make false")
			}
			if !strings.Contains(body, "not stored") {
				t.Errorf("page should still state the file is not stored")
			}
			if !strings.Contains(body, "your browser") && !strings.Contains(body, "your browser</strong>") {
				t.Errorf("page should explain that map backgrounds are loaded by the browser")
			}
		})
	}
}

// TestNoRestrictiveReferrerPolicy protects tile access: the OSM Foundation
// policy requires web applications to send a valid Referer, and stripping it
// risks being blocked without notice.
func TestNoRestrictiveReferrerPolicy(t *testing.T) {
	srv := testServer(t, nil)
	for _, rec := range []string{"/"} {
		resp := serve(t, srv, getRequest(t, rec))
		switch got := resp.Header().Get("Referrer-Policy"); got {
		case "", "strict-origin-when-cross-origin", "no-referrer-when-downgrade", "origin-when-cross-origin":
			// Fine: the provider still receives a Referer.
		default:
			t.Errorf("Referrer-Policy = %q, which may strip the Referer that OSM requires", got)
		}
	}
}

// TestNoExternalAssetReferences guards the self-contained-artefact rule: every
// script and stylesheet must be served by us, never by a CDN.
func TestNoExternalAssetReferences(t *testing.T) {
	srv := testServer(t, nil)
	pages := map[string]string{
		"index":   serve(t, srv, getRequest(t, "/")).Body.String(),
		"results": serve(t, srv, uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil)).Body.String(),
	}
	// Only <script src> and <link href> are loaded assets. Attribution links
	// live inside the JSON payload with escaped quotes (href=\"...\"), so they
	// do not match this pattern and are correctly ignored: they are links the
	// licence requires, not resources the page fetches.
	asset := regexp.MustCompile(`<(?:script|link)\b[^>]*?(?:src|href)="([^"]+)"`)
	for name, body := range pages {
		t.Run(name, func(t *testing.T) {
			refs := asset.FindAllStringSubmatch(body, -1)
			if len(refs) == 0 {
				t.Fatalf("found no script/link assets; the pattern is not matching anything")
			}
			for _, m := range refs {
				if ref := m[1]; !strings.HasPrefix(ref, "/") {
					t.Errorf("page loads off-origin asset %q; the artefact must be self-contained", ref)
				}
			}
		})
	}
}

func mustParse(t *testing.T, data []byte) gpx.Track {
	t.Helper()
	tr, err := gpx.Parse(strings.NewReader(string(data)), 1<<20, 100000)
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	return tr
}
