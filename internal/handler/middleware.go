package handler

import (
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/nikitaw13/metricscollector/internal/model"
	"go.uber.org/zap"
)

// requireJSONContent enforces application/json Content-Type and non-empty body
// for endpoints that expect JSON payloads.
func requireJSONContent(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			http.Error(w, "Type is required", http.StatusBadRequest)
			return
		}
		if r.ContentLength == 0 {
			writeJSONError(w, http.StatusBadRequest, errors.New("request body is required"))
			return
		}
		h.ServeHTTP(w, r)
	})
}

// requireMetricType rejects requests with an unknown metric type.
func requireMetricType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "TYPE")
		if metricType != model.Gauge && metricType != model.Counter {
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggerMiddleware logs incoming HTTP requests and their responses.
func loggerMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		respInfo := &responseInfo{
			status: 0,
			size:   0,
		}

		loggingWriter := loggingResponseWriter{
			ResponseWriter: w,
			responseInfo:   respInfo,
		}

		h.ServeHTTP(&loggingWriter, r)

		Logger.Info("got incoming HTTP request",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Duration("duration", time.Since(start)),
		)

		Logger.Info("response for incoming HTTP request",
			zap.Int("status", respInfo.status),
			zap.Int("size", respInfo.size),
		)
	})
}

// compressibleTypes defines Content-Type values eligible for compression.
var compressibleTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

// compressWriter routes writes through an encoder (gzip/deflate).
// Non-compressible content types bypass the encoder entirely.
type compressWriter struct {
	http.ResponseWriter
	Writer   io.WriteCloser // encoder instance
	encoding string         // "gzip" or "deflate"
	bypassed bool           // set when bypassing; prevents encoder Close
}

// WriteHeader checks Content-Type before headers are finalized.
// Sets Content-Encoding only for compressible types; marks non-compressible as bypassed.
func (c *compressWriter) WriteHeader(code int) {
	ct := c.ResponseWriter.Header().Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(ct)
	if compressibleTypes[mediaType] {
		c.ResponseWriter.Header().Set("Content-Encoding", c.encoding)
	} else {
		c.bypassed = true
	}
	c.ResponseWriter.WriteHeader(code)
}

// Write compresses data for eligible Content-Type; otherwise writes directly
// to the underlying ResponseWriter.
func (c *compressWriter) Write(b []byte) (int, error) {
	if c.bypassed {
		return c.ResponseWriter.Write(b)
	}
	ct := c.ResponseWriter.Header().Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(ct)
	if !compressibleTypes[mediaType] {
		c.bypassed = true
		c.ResponseWriter.Header().Del("Content-Encoding")
		return c.ResponseWriter.Write(b)
	}
	if c.ResponseWriter.Header().Get("Content-Encoding") == "" {
		c.ResponseWriter.Header().Set("Content-Encoding", c.encoding)
	}
	return c.Writer.Write(b)
}

// CompressMiddleware compresses response bodies for clients supporting
// Accept-Encoding. Only application/json and text/html are compressed.
// Gzip takes priority over deflate.
func CompressMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		acceptEncoding := r.Header.Get("Accept-Encoding")

		if strings.Contains(acceptEncoding, "gzip") {
			// BestCompression is a package constant, so NewWriterLevel never errors here.
			gz, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
			compressWriter := &compressWriter{ResponseWriter: w, Writer: gz, encoding: "gzip"}
			h.ServeHTTP(compressWriter, r)
			if !compressWriter.bypassed {
				gz.Close()
			}
			return
		}

		if strings.Contains(acceptEncoding, "deflate") {
			// BestCompression is a package constant, so NewWriter never errors here.
			dw, _ := flate.NewWriter(w, flate.BestCompression)
			compressWriter := &compressWriter{ResponseWriter: w, Writer: dw, encoding: "deflate"}
			h.ServeHTTP(compressWriter, r)
			if !compressWriter.bypassed {
				dw.Close()
			}
			return
		}

		h.ServeHTTP(w, r)
	})
}

// DecompressMiddleware decompresses gzip/deflate request bodies based on
// Content-Encoding. Replaces r.Body and removes encoding/length headers.
func DecompressMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defer gz.Close()
			r.Header.Del("Content-Encoding")
			r.Header.Del("Content-Length")
			r.ContentLength = -1
			r.Body = io.NopCloser(gz)
		case "deflate":
			df := flate.NewReader(r.Body)
			defer df.Close()
			r.Header.Del("Content-Encoding")
			r.Header.Del("Content-Length")
			r.ContentLength = -1
			r.Body = io.NopCloser(df)
		}

		h.ServeHTTP(w, r)
	})
}
