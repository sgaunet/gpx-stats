package web

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// Compression exists for one reason: the map feature embeds every track point
// in the results page unreduced (FR-001a), which is ~9.5 MB of coordinates for
// the largest track the parser accepts. Numeric JSON compresses about 4-5x, so
// gzip takes that worst case to roughly 2 MB on the wire, and a typical ride to
// about 55 KB. It also shrinks the inline SVG charts.
//
// It reduces transfer only — the browser still parses and renders every point.
// That residual cost is the accepted trade recorded in the feature spec.

// gzipResponseWriter compresses the body while leaving status and header
// semantics untouched. Header rewriting happens once, on the first write, so
// that handlers which set headers after WriteHeader still behave correctly.
type gzipResponseWriter struct {
	http.ResponseWriter

	zw          *gzip.Writer
	wroteHeader bool
	compressing bool
}

// WriteHeader decides whether this particular response gets compressed. Only
// text-ish content types are worth it; anything already compressed is passed
// through untouched.
func (w *gzipResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	if shouldCompress(w.Header().Get("Content-Type")) {
		w.compressing = true
		w.Header().Set("Content-Encoding", "gzip")
		// The compressed length is not known up front, and a stale value would
		// truncate the response.
		w.Header().Del("Content-Length")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		// Mirror net/http: an unheadered write implies 200.
		w.WriteHeader(http.StatusOK)
	}
	if !w.compressing {
		n, err := w.ResponseWriter.Write(b) //nolint:wrapcheck // pass-through writer
		return n, err
	}
	n, err := w.zw.Write(b) //nolint:wrapcheck // pass-through writer
	return n, err
}

// Flush forwards flushes so streaming handlers keep working. The gzip writer is
// flushed first, otherwise buffered bytes would arrive after the flush.
func (w *gzipResponseWriter) Flush() {
	if w.compressing {
		_ = w.zw.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// shouldCompress reports whether a response of this content type benefits from
// compression. Images, fonts and archives are already compressed; re-deflating
// them burns CPU to make them marginally larger.
func shouldCompress(contentType string) bool {
	if contentType == "" {
		return false
	}
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))

	switch {
	case strings.HasPrefix(mediaType, "text/"):
		return true
	case mediaType == "application/json",
		mediaType == "application/javascript",
		mediaType == "application/xml",
		mediaType == "image/svg+xml":
		return true
	default:
		return false
	}
}

// gzipMiddleware compresses responses for clients that advertise gzip support.
// Clients that do not receive byte-identical responses to before.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vary is set regardless of whether this particular response ends up
		// compressed: the response body varies by Accept-Encoding, so shared
		// caches must key on it either way.
		w.Header().Add("Vary", "Accept-Encoding")

		if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		zw := gzip.NewWriter(w)
		gw := &gzipResponseWriter{ResponseWriter: w, zw: zw}
		// Closing flushes the gzip trailer. It must happen before the handler
		// chain returns, or the client receives a truncated stream.
		defer func() {
			if gw.compressing {
				_ = zw.Close()
			}
		}()

		next.ServeHTTP(gw, r)
	})
}

// acceptsGzip reports whether the Accept-Encoding header offers gzip. It looks
// for gzip as a whole token so that encodings like "x-gzip-ish" do not match.
func acceptsGzip(header string) bool {
	for enc := range strings.SplitSeq(header, ",") {
		token, _, _ := strings.Cut(enc, ";")
		if strings.EqualFold(strings.TrimSpace(token), "gzip") {
			return true
		}
	}
	return false
}
