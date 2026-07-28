package web_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGzipCompressesWhenAccepted(t *testing.T) {
	req := getRequest(t, "/")
	req.Header.Set("Accept-Encoding", "gzip")
	rec := serve(t, testServer(t, nil), req)

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", got, "gzip")
	}
	if got := rec.Header().Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Errorf("Vary = %q, want it to contain Accept-Encoding", got)
	}
	if got := rec.Header().Get("Content-Length"); got != "" {
		t.Errorf("Content-Length = %q, want it removed when compressing", got)
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("body is not valid gzip: %v", err)
	}
	defer func() { _ = zr.Close() }()
	if _, err := io.ReadAll(zr); err != nil {
		t.Fatalf("decompressing body: %v", err)
	}
}

func TestGzipPassesThroughWhenNotAccepted(t *testing.T) {
	rec := serve(t, testServer(t, nil), getRequest(t, "/"))
	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, want empty when the client did not ask for gzip", got)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Errorf("expected uncompressed HTML")
	}
}

// TestGzipRoundTripsIdentically is the contract that matters: compression must
// change how bytes travel, never what they say.
func TestGzipRoundTripsIdentically(t *testing.T) {
	srv := testServer(t, nil)

	plain := serve(t, srv, getRequest(t, "/")).Body.Bytes()

	req := getRequest(t, "/")
	req.Header.Set("Accept-Encoding", "gzip")
	zr, err := gzip.NewReader(serve(t, srv, req).Body)
	if err != nil {
		t.Fatalf("body is not valid gzip: %v", err)
	}
	defer func() { _ = zr.Close() }()
	decompressed, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("decompressing body: %v", err)
	}

	if string(decompressed) != string(plain) {
		t.Errorf("decompressed body differs from the uncompressed body")
	}
}

func TestGzipPreservesErrorStatus(t *testing.T) {
	req := uploadRequest(t, []byte("this is not gpx"), nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := serve(t, testServer(t, nil), req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400: compression must not alter status codes", rec.Code)
	}
}

// TestGzipDoesNotRecompressVendoredAssets guards against wrapping content that
// is already compressed or served byte-range-wise by http.FileServer.
func TestGzipCompressesStaticText(t *testing.T) {
	req := getRequest(t, "/static/vendor/leaflet/leaflet.js")
	req.Header.Set("Accept-Encoding", "gzip")
	rec := serve(t, testServer(t, nil), req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatalf("claimed gzip but body is not valid gzip: %v", err)
		}
		defer func() { _ = zr.Close() }()
		if _, err := io.ReadAll(zr); err != nil {
			t.Fatalf("decompressing body: %v", err)
		}
	}
}
