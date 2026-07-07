package cli

import (
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/web"
)

// runServe implements `gpx-stats serve [flags]`, starting the embedded web UI.
func runServe(args []string, _ io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("gpx-stats serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: gpx-stats serve [flags]")
		fmt.Fprintln(stderr, "\nflags:")
		fs.PrintDefaults()
	}

	addr := fs.String("addr", config.DefaultServerAddr, "listen address")
	maxUpload := fs.Int64("max-upload", config.DefaultMaxUploadBytes, "maximum upload size in bytes")
	maxPoints := fs.Int("max-points", config.DefaultMaxTrackPoints, "maximum track points accepted")
	pauseSpeed := fs.Float64("pause-speed", config.DefaultStationarySpeedKmh, "default stationary speed threshold in km/h")
	pauseDur := fs.Duration("pause-duration", config.DefaultMinPauseDuration, "default minimum pause duration")

	if code, done := parseFlags(fs, args, stderr); done {
		return code
	}

	cfg := config.Default()
	cfg.ServerAddr = *addr
	cfg.MaxUploadBytes = *maxUpload
	cfg.MaxTrackPoints = *maxPoints
	cfg.StationarySpeedKmh = *pauseSpeed
	cfg.MinPauseDuration = *pauseDur
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(stderr, "error: invalid configuration: %v\n", err)
		return exitUsage
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	srv, err := web.NewServer(cfg, logger)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitRuntime
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitRuntime
	}
	return exitOK
}
