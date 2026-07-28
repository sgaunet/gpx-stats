// Package config holds the typed, centralized configuration for gpx-stats.
//
// Configuration is explicit and passed to the statistics engine and transports
// rather than read from package-level globals (project constitution: explicit,
// typed configuration).
package config

import (
	"fmt"
	"time"
)

// Default values for configuration fields.
const (
	// DefaultStationarySpeedKmh is deliberately small: it tolerates the GPS
	// jitter a stopped device still records, without counting slow movement as
	// a stop. Set it to 0 to require a true standstill.
	DefaultStationarySpeedKmh   = 0.1
	DefaultMinPauseDuration     = 120 * time.Second
	DefaultElevationNoiseMeters = 1.0
	DefaultMaxUploadBytes       = int64(25 << 20) // 25 MiB
	DefaultMaxTrackPoints       = 500_000
	DefaultServerAddr           = ":8080"
)

// Config carries every tunable parameter for computing statistics and running
// the web server. Construct with Default and override fields as needed, then
// call Validate before use.
type Config struct {
	// StationarySpeedKmh is the speed at or below which a segment is considered
	// stationary for pause detection. Zero requires a true standstill.
	StationarySpeedKmh float64
	// MinPauseDuration is the minimum length of a stationary run to count as a
	// pause.
	MinPauseDuration time.Duration
	// ElevationNoiseMeters is the hysteresis threshold used to filter GPS
	// elevation jitter when summing ascending gain.
	ElevationNoiseMeters float64
	// MaxUploadBytes bounds the size of a GPX input (defense against memory
	// exhaustion).
	MaxUploadBytes int64
	// MaxTrackPoints bounds the number of track points accepted from an input.
	MaxTrackPoints int
	// ServerAddr is the listen address used by the web server.
	ServerAddr string
}

// Default returns a Config populated with the documented default values.
func Default() Config {
	return Config{
		StationarySpeedKmh:   DefaultStationarySpeedKmh,
		MinPauseDuration:     DefaultMinPauseDuration,
		ElevationNoiseMeters: DefaultElevationNoiseMeters,
		MaxUploadBytes:       DefaultMaxUploadBytes,
		MaxTrackPoints:       DefaultMaxTrackPoints,
		ServerAddr:           DefaultServerAddr,
	}
}

// Validate reports the first invalid field, if any, with an actionable message.
func (c Config) Validate() error {
	if c.StationarySpeedKmh < 0 {
		return fmt.Errorf("stationary speed must be >= 0 km/h, got %g", c.StationarySpeedKmh)
	}
	if c.MinPauseDuration <= 0 {
		return fmt.Errorf("minimum pause duration must be > 0, got %s", c.MinPauseDuration)
	}
	if c.ElevationNoiseMeters < 0 {
		return fmt.Errorf("elevation noise threshold must be >= 0 m, got %g", c.ElevationNoiseMeters)
	}
	if c.MaxUploadBytes <= 0 {
		return fmt.Errorf("max upload bytes must be > 0, got %d", c.MaxUploadBytes)
	}
	if c.MaxTrackPoints <= 0 {
		return fmt.Errorf("max track points must be > 0, got %d", c.MaxTrackPoints)
	}
	return nil
}
