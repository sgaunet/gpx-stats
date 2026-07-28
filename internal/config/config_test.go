package config_test

import (
	"testing"
	"time"

	"github.com/sgaunet/gpx-stats/internal/config"
)

func TestDefaultIsValid(t *testing.T) {
	if err := config.Default().Validate(); err != nil {
		t.Fatalf("Default() should be valid, got: %v", err)
	}
	d := config.Default()
	if d.StationarySpeedKmh != 0.1 {
		t.Errorf("default stationary speed = %g, want 0.1", d.StationarySpeedKmh)
	}
	if d.MinPauseDuration != 120*time.Second {
		t.Errorf("default min pause = %s, want 2m0s", d.MinPauseDuration)
	}
	if d.ElevationNoiseMeters != 1.0 {
		t.Errorf("default elevation noise = %g, want 1.0", d.ElevationNoiseMeters)
	}
	if d.MaxUploadBytes != 25<<20 {
		t.Errorf("default max upload = %d, want %d", d.MaxUploadBytes, 25<<20)
	}
	if d.MaxTrackPoints != 500_000 {
		t.Errorf("default max points = %d, want 500000", d.MaxTrackPoints)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *config.Config)
		wantErr bool
	}{
		{"default", func(*config.Config) {}, false},
		{"zero stationary speed allowed", func(c *config.Config) { c.StationarySpeedKmh = 0 }, false},
		{"negative stationary speed", func(c *config.Config) { c.StationarySpeedKmh = -1 }, true},
		{"zero pause duration", func(c *config.Config) { c.MinPauseDuration = 0 }, true},
		{"negative pause duration", func(c *config.Config) { c.MinPauseDuration = -5 }, true},
		{"negative elevation noise", func(c *config.Config) { c.ElevationNoiseMeters = -0.1 }, true},
		{"zero max upload", func(c *config.Config) { c.MaxUploadBytes = 0 }, true},
		{"negative max points", func(c *config.Config) { c.MaxTrackPoints = -10 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.Default()
			tt.mutate(&c)
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
