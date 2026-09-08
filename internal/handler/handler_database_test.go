package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	handlermock "github.com/nikitaw13/metricscollector/internal/handler/mock"
	"github.com/nikitaw13/metricscollector/internal/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestPing verifies that /ping returns "Pong" on success — via a mock
// database or an in-memory storage wired as the pinger — and 500 when the
// database is unreachable.
func TestPing(t *testing.T) {
	t.Run("Successful /ping", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDB := handlermock.NewMockDBPinger(ctrl)
		ts := GetTestServerWithDatabase(mockDB)
		defer ts.Close()
		mockDB.EXPECT().PingContext(gomock.Any()).Return(nil)

		resp := doRequest(t, ts, http.MethodGet, "/ping")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		require.Equal(t, "Pong\n", resp.body)
	})

	t.Run("Successful /ping with in-memory storage", func(t *testing.T) {
		t.Parallel()
		storage := repository.NewMemStorage()
		ts := httptest.NewServer(NewMetricsHandler(storage, storage).NewRouter())
		defer ts.Close()

		resp := doRequest(t, ts, http.MethodGet, "/ping")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		require.Equal(t, "Pong\n", resp.body)
	})

	t.Run("Unsuccessful /ping", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDB := handlermock.NewMockDBPinger(ctrl)
		ts := GetTestServerWithDatabase(mockDB)
		defer ts.Close()
		mockDB.EXPECT().PingContext(gomock.Any()).Return(errors.New("connection refused"))

		resp := doRequest(t, ts, http.MethodGet, "/ping")
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
		require.Equal(t, "Failed to ping database\n", resp.body)
	})
}
