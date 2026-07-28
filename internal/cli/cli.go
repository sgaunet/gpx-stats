package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
)

// Exit codes (see contracts/cli.md).
const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2
)

// Run parses args (excluding the program name) and executes the requested
// subcommand, writing normal output to stdout and errors to stderr. It returns
// a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "serve" {
		return runServe(args[1:], stdout, stderr)
	}
	return runStats(args, stdout, stderr)
}

// runStats implements `gpx-stats [flags] <path>`.
func runStats(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gpx-stats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: gpx-stats [flags] <path-to-gpx>")
		fmt.Fprintln(stderr, "       gpx-stats serve [flags]")
		fmt.Fprintln(stderr, "\nflags:")
		fs.PrintDefaults()
	}

	jsonOut := fs.Bool("json", false, "emit statistics as JSON instead of text")
	charts := fs.Bool("charts", false, "also draw ASCII elevation and speed charts")
	pauseSpeed := fs.Float64("pause-speed", config.DefaultStationarySpeedKmh,
		"stationary speed threshold in km/h (a segment at or below it counts as stopped)")
	pauseDur := fs.Duration("pause-duration", config.DefaultMinPauseDuration, "minimum pause duration")
	elevNoise := fs.Float64("elevation-noise", config.DefaultElevationNoiseMeters,
		"elevation jitter threshold in meters (filters GPS noise out of ascent and descent)")
	maxPoints := fs.Int("max-points", config.DefaultMaxTrackPoints, "maximum track points accepted")

	if code, done := parseFlags(fs, args, stderr); done {
		return code
	}

	rest := fs.Args()
	if len(rest) != 1 {
		fs.Usage()
		return exitUsage
	}
	path := rest[0]

	cfg := config.Default()
	cfg.StationarySpeedKmh = *pauseSpeed
	cfg.MinPauseDuration = *pauseDur
	cfg.ElevationNoiseMeters = *elevNoise
	cfg.MaxTrackPoints = *maxPoints
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(stderr, "error: invalid configuration: %v\n", err)
		return exitUsage
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: cannot open %s: %v\n", path, err)
		return exitRuntime
	}
	defer func() { _ = f.Close() }()

	track, err := gpx.Parse(f, cfg.MaxUploadBytes, cfg.MaxTrackPoints)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitRuntime
	}

	res := stats.Compute(track, cfg)
	if *jsonOut {
		if err := WriteJSON(stdout, res); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return exitRuntime
		}
		return exitOK
	}
	WriteText(stdout, res)
	if *charts {
		WriteCharts(stdout, track, res)
	}
	return exitOK
}

// parseFlags parses args and distinguishes a help request (exit 0) from a usage
// error (exit 2). done is true when the caller should return the returned code.
func parseFlags(fs *flag.FlagSet, args []string, _ io.Writer) (code int, done bool) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK, true
		}
		return exitUsage, true
	}
	return exitOK, false
}
