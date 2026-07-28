package web_test

import (
	"strings"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/web"
)

// The base-layer table carries obligations that are easy to break silently: a
// missing attribution breaches the OSM Foundation tile usage policy, and a
// plaintext URL would trigger mixed-content blocking. These assert them.

func TestBaseLayersHaveExactlyOneDefault(t *testing.T) {
	var defaults []string
	for _, l := range web.BaseLayers {
		if l.Default {
			defaults = append(defaults, l.Key)
		}
	}
	if len(defaults) != 1 {
		t.Errorf("found %d default layers (%v), want exactly 1", len(defaults), defaults)
	}
}

func TestBaseLayersKeysAreUniqueAndNonEmpty(t *testing.T) {
	seen := make(map[string]bool, len(web.BaseLayers))
	for i, l := range web.BaseLayers {
		if l.Key == "" {
			t.Errorf("layer %d has an empty key", i)
		}
		if seen[l.Key] {
			t.Errorf("duplicate layer key %q", l.Key)
		}
		seen[l.Key] = true
		if l.Name == "" {
			t.Errorf("layer %q has an empty name", l.Key)
		}
	}
}

func TestBaseLayersUseHTTPSOnly(t *testing.T) {
	for _, l := range web.BaseLayers {
		if !strings.HasPrefix(l.URLTemplate, "https://") {
			t.Errorf("layer %q URL %q is not HTTPS: tiles would be blocked as mixed content",
				l.Key, l.URLTemplate)
		}
	}
}

// TestBaseLayersCreditOpenStreetMap enforces the attribution obligation. Every
// offered layer is built on OSM data, so every one must credit it.
func TestBaseLayersCreditOpenStreetMap(t *testing.T) {
	for _, l := range web.BaseLayers {
		if l.Attribution == "" {
			t.Errorf("layer %q has no attribution", l.Key)
			continue
		}
		if !strings.Contains(l.Attribution, "OpenStreetMap") {
			t.Errorf("layer %q attribution does not credit OpenStreetMap: %q", l.Key, l.Attribution)
		}
	}
}

func TestBaseLayersMaxZoomInRange(t *testing.T) {
	for _, l := range web.BaseLayers {
		if l.MaxZoom < 1 || l.MaxZoom > 22 {
			t.Errorf("layer %q MaxZoom = %d, want between 1 and 22", l.Key, l.MaxZoom)
		}
	}
}

// TestBaseLayersOfferStreetAndTopo checks the spec's minimum: at least three
// layers, including a general-purpose street map and a topographic one.
func TestBaseLayersOfferStreetAndTopo(t *testing.T) {
	if len(web.BaseLayers) < 3 {
		t.Fatalf("only %d base layers, want at least 3", len(web.BaseLayers))
	}
	var hasStreet, hasTopo bool
	for _, l := range web.BaseLayers {
		switch l.Key {
		case "osm":
			hasStreet = true
		case "topo":
			hasTopo = true
		}
	}
	if !hasStreet {
		t.Errorf("no general-purpose street map layer offered")
	}
	if !hasTopo {
		t.Errorf("no topographic layer offered")
	}
}
