// Package web implements the HTTP transport for gpx-stats: a stateless,
// single-user web UI that accepts a GPX upload, computes statistics, and renders
// them with inline SVG charts. It stores nothing and makes no outbound calls.
// All templates and static assets are embedded in the binary.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/sgaunet/gpx-stats/internal/config"
)

//go:embed assets/templates/*.html
var templatesFS embed.FS

//go:embed assets/static/*
var staticFS embed.FS

// Timeouts for the HTTP server (Constitution: timeouts on all I/O).
const (
	readTimeout       = 15 * time.Second
	readHeaderTimeout = 5 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 1 << 20
)

// Server serves the gpx-stats web UI.
type Server struct {
	cfg  config.Config
	log  *slog.Logger
	tmpl *template.Template
}

// NewServer parses the embedded templates and returns a ready Server.
func NewServer(cfg config.Config, log *slog.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	tmpl, err := template.ParseFS(templatesFS, "assets/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	return &Server{cfg: cfg, log: log, tmpl: tmpl}, nil
}

// Handler returns the HTTP handler for the server (exposed for testing).
func (s *Server) Handler() http.Handler {
	staticSub, err := fs.Sub(staticFS, "assets/static")
	if err != nil {
		// Embedded FS: this can only fail on a programming error.
		panic(err)
	}
	mux := http.NewServeMux()
	// {$} anchors the pattern to exactly "/" so that other paths (e.g. a GET to
	// /analyze) fall through to a 405 method-mismatch rather than the form.
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("POST /analyze", s.handleAnalyze)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))
	return mux
}

// ListenAndServe starts the server with sane timeouts.
func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:              s.cfg.ServerAddr,
		Handler:           s.Handler(),
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
	s.log.Info("starting gpx-stats web server", "addr", s.cfg.ServerAddr)
	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
