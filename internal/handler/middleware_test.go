package handler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedResponseHandler returns a handler replying with the given Content-Type and body.
func fixedResponseHandler(contentType, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(body))
	})
}

// TestCompressMiddlewareGzipResponse verifies that responses are gzip-compressed for gzip-capable clients.
func TestCompressMiddlewareGzipResponse(t *testing.T) {
	handler := CompressMiddleware(fixedResponseHandler("application/json", `{"status":"ok"}`))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gzipReader, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer gzipReader.Close()

	body, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	assert.Equal(t, `{"status":"ok"}`, string(body))
}

// TestCompressMiddlewareDeflateResponse verifies deflate compression when only deflate is accepted.
func TestCompressMiddlewareDeflateResponse(t *testing.T) {
	handler := CompressMiddleware(fixedResponseHandler("application/json", `{"status":"ok"}`))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "deflate")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "deflate", rec.Header().Get("Content-Encoding"))

	flateReader := flate.NewReader(rec.Body)
	defer flateReader.Close()

	body, err := io.ReadAll(flateReader)
	require.NoError(t, err)
	assert.Equal(t, `{"status":"ok"}`, string(body))
}

// TestCompressMiddlewareNoCompressionWithoutAcceptEncoding verifies the uncompressed passthrough when no encoding is accepted.
func TestCompressMiddlewareNoCompressionWithoutAcceptEncoding(t *testing.T) {
	handler := CompressMiddleware(fixedResponseHandler("application/json", `{"status":"ok"}`))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, `{"status":"ok"}`, rec.Body.String())
}

// TestCompressMiddlewareMultipleAcceptEncoding verifies that gzip wins when several encodings are accepted.
func TestCompressMiddlewareMultipleAcceptEncoding(t *testing.T) {
	handler := CompressMiddleware(fixedResponseHandler("application/json", `{"status":"ok"}`))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
}

// TestCompressMiddlewareSkipsNonCompressibleType verifies that text/plain responses bypass compression.
func TestCompressMiddlewareSkipsNonCompressibleType(t *testing.T) {
	handler := CompressMiddleware(fixedResponseHandler("text/plain", "hello"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "hello", rec.Body.String())
}

// TestCompressMiddlewareCompressesHTML verifies that text/html responses are gzip-compressed.
func TestCompressMiddlewareCompressesHTML(t *testing.T) {
	handler := CompressMiddleware(fixedResponseHandler("text/html; charset=utf-8", "<html>hi</html>"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gzipReader, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	defer gzipReader.Close()

	body, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	assert.Equal(t, "<html>hi</html>", string(body))
}

// TestDecompressMiddlewareGzipBody verifies that gzip request bodies are decompressed before reaching the handler.
func TestDecompressMiddlewareGzipBody(t *testing.T) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write([]byte(`{"status":"ok"}`))
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	var gotBody string
	handler := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"ok"}`, gotBody)
	assert.Empty(t, req.Header.Get("Content-Encoding"))
}

// TestDecompressMiddlewareDeflateBody verifies that deflate request bodies are decompressed before reaching the handler.
func TestDecompressMiddlewareDeflateBody(t *testing.T) {
	var buf bytes.Buffer
	flateWriter, _ := flate.NewWriter(&buf, flate.BestCompression)
	_, err := flateWriter.Write([]byte(`{"status":"ok"}`))
	require.NoError(t, err)
	require.NoError(t, flateWriter.Close())

	var gotBody string
	handler := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "deflate")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"ok"}`, gotBody)
	assert.Empty(t, req.Header.Get("Content-Encoding"))
}

// TestDecompressMiddlewareNoEncodingPassthrough verifies that raw bodies pass through unchanged.
func TestDecompressMiddlewareNoEncodingPassthrough(t *testing.T) {
	var gotBody string
	handler := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("raw body"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "raw body", gotBody)
}

// TestDecompressMiddlewareInvalidGzipBody verifies that a broken gzip body is rejected with 500.
func TestDecompressMiddlewareInvalidGzipBody(t *testing.T) {
	handler := DecompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestRequireJSONContentEmptyBody verifies that an empty JSON request body is rejected with 400.
func TestRequireJSONContentEmptyBody(t *testing.T) {
	handler := requireJSONContent(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
