package web_test

import (
	"bytes"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgaunet/gpx-stats/internal/config"
	"github.com/sgaunet/gpx-stats/internal/web"
)

func testServer(t *testing.T, mutate func(c *config.Config)) *web.Server {
	t.Helper()
	cfg := config.Default()
	if mutate != nil {
		mutate(&cfg)
	}
	srv, err := web.NewServer(cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv
}

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return b
}

func getRequest(t *testing.T, target string) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
}

func uploadRequest(t *testing.T, content []byte, fields map[string]string) *http.Request {
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
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/analyze", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func serve(t *testing.T, srv *web.Server, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestIndexShowsFormAndPrivacy(t *testing.T) {
	rec := serve(t, testServer(t, nil), getRequest(t, "/"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "not stored") {
		t.Errorf("index should state the file is not stored")
	}
	if !strings.Contains(body, `action="/analyze"`) {
		t.Errorf("index should contain the upload form")
	}
}

func TestAnalyzeValid(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Activity statistics") {
		t.Errorf("expected results page")
	}
	if !strings.Contains(body, "<svg") {
		t.Errorf("expected an inline SVG chart in the results")
	}
}

func TestAnalyzeInvalidFile(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, []byte("this is not gpx"), nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Could not") {
		t.Errorf("expected an actionable error message")
	}
}

// TestAnalyzeMalicious is the security case: an XXE / entity-expansion GPX must
// be rejected with 400 and no crash, file read, or leak.
func TestAnalyzeMalicious(t *testing.T) {
	rec := serve(t, testServer(t, nil), uploadRequest(t, fixtureBytes(t, "malicious.gpx"), nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "root:") {
		t.Fatalf("response appears to contain local file contents (XXE leak)")
	}
}

func TestAnalyzeTooLarge(t *testing.T) {
	srv := testServer(t, func(c *config.Config) { c.MaxUploadBytes = 10 })
	rec := serve(t, srv, uploadRequest(t, fixtureBytes(t, "sample.gpx"), nil))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
}

func TestAnalyzeMissingFileField(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("pause_speed", "1.0"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/analyze", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := serve(t, testServer(t, nil), req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAnalyzeInvalidPauseField(t *testing.T) {
	req := uploadRequest(t, fixtureBytes(t, "sample.gpx"), map[string]string{"pause_duration": "notaduration"})
	rec := serve(t, testServer(t, nil), req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAnalyzeRejectsGET(t *testing.T) {
	rec := serve(t, testServer(t, nil), getRequest(t, "/analyze"))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestStaticAssetServed(t *testing.T) {
	rec := serve(t, testServer(t, nil), getRequest(t, "/static/bulma.min.css"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("content-type = %q, want css", ct)
	}
}
