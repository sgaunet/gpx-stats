package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/cli"
	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/gpx"
	"github.com/sgaunet/gpx-stats/internal/stats"
	"github.com/sgaunet/gpx-stats/internal/web"
)

// TestCLIWebParity verifies that the CLI (--json) and the web UI report the same
// statistics for the same input (SC-005). Both derive from stats.Compute, so
// this guards against a transport diverging from the shared engine.
func TestCLIWebParity(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.gpx")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	cfg := config.Default()

	// Canonical result computed directly from the engine.
	track, err := gpx.Parse(bytes.NewReader(raw), cfg.MaxUploadBytes, cfg.MaxTrackPoints)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := stats.Compute(track, cfg)

	// CLI JSON path.
	var out, errOut bytes.Buffer
	if code := cli.Run([]string{"--json", path}, &out, &errOut); code != 0 {
		t.Fatalf("cli exit %d: %s", code, errOut.String())
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("cli json: %v", err)
	}
	if d := floatField(t, got, "totalDistanceKm"); math.Abs(d-round3(want.TotalDistanceKm)) > 1e-9 {
		t.Errorf("cli distance %.3f != engine %.3f", d, want.TotalDistanceKm)
	}
	if s := floatField(t, got, "totalTimeSeconds"); s != want.TotalTime.Seconds() {
		t.Errorf("cli total time %.0f != engine %.0f", s, want.TotalTime.Seconds())
	}

	// Web path: the results page must display the same distance (2-decimal).
	srv, err := web.NewServer(cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("web server: %v", err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, uploadReq(t, raw))
	if rec.Code != http.StatusOK {
		t.Fatalf("web status %d", rec.Code)
	}
	wantDistStr := fmt.Sprintf("%.2f km", want.TotalDistanceKm)
	if !strings.Contains(rec.Body.String(), wantDistStr) {
		t.Errorf("web page missing distance %q", wantDistStr)
	}
}

// round3 mirrors the CLI's JSON rounding for comparison.
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

func uploadReq(t *testing.T, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "activity.gpx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/analyze", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}
