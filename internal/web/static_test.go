package web_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestVendoredAssetsAreServed proves the existing //go:embed assets/static/*
// directive recurses into subdirectories, so the vendored Leaflet files ship
// inside the binary without an embed-directive change (research.md R2).
func TestVendoredAssetsAreServed(t *testing.T) {
	srv := testServer(t, nil)
	for _, path := range []string{
		"/static/vendor/leaflet/leaflet.js",
		"/static/vendor/leaflet/leaflet.css",
		"/static/vendor/leaflet/LICENSE",
		"/static/map.js",
		"/static/map.css",
	} {
		t.Run(path, func(t *testing.T) {
			rec := serve(t, srv, getRequest(t, path))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want 200", path, rec.Code)
			}
			if rec.Body.Len() == 0 {
				t.Errorf("GET %s returned an empty body", path)
			}
		})
	}
}

// TestNoClientSidePersistence guards the promise that nothing about a track is
// kept anywhere. Remembering the selected base layer in localStorage is the
// obvious "free" improvement someone will reach for; the spec rules it out,
// and nothing but this test would catch it.
func TestNoClientSidePersistence(t *testing.T) {
	srv := testServer(t, nil)

	script := serve(t, srv, getRequest(t, "/static/map.js")).Body.String()
	for _, forbidden := range []string{
		"localStorage",
		"sessionStorage",
		"indexedDB",
		"document.cookie",
	} {
		if strings.Contains(script, forbidden) {
			t.Errorf("map.js uses %s: map view state must never be persisted", forbidden)
		}
	}

	t.Run("no cookies set", func(t *testing.T) {
		for _, rec := range []*httpRecorder{
			{"index", serve(t, srv, getRequest(t, "/")).Header()},
			{"results", serve(t, srv,
				uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil)).Header()},
		} {
			if got := rec.header.Get("Set-Cookie"); got != "" {
				t.Errorf("%s response sets a cookie (%q); the app is stateless", rec.name, got)
			}
		}
	})
}

type httpRecorder struct {
	name   string
	header http.Header
}

// TestNoScriptFallback confirms the page degrades rather than breaks: the map
// is the only thing that needs scripting, and the statistics are all rendered
// server-side.
func TestNoScriptFallback(t *testing.T) {
	srv := testServer(t, nil)
	body := serve(t, srv, uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil)).Body.String()

	if !strings.Contains(body, "<noscript>") {
		t.Errorf("results page should carry a <noscript> note in place of the map")
	}
	for _, want := range []string{"Activity statistics", "Kilometer splits", "<svg"} {
		if !strings.Contains(body, want) {
			t.Errorf("statistics content %q must render without scripting", want)
		}
	}
}
