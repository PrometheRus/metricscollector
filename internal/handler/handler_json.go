package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nikitaw13/metricscollector/internal/model"
	"go.uber.org/zap"
)

// apiError holds the JSON body for error responses.
type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// decodeError carries a client-safe message while preserving the original
// error chain for errors.Is/As and logging.
type decodeError struct {
	status  int
	message string
	err     error
}

func (e *decodeError) Error() string { return e.message }

func (e *decodeError) Unwrap() error { return e.err }

// writeJSON marshals payload and writes it as the response body with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal error"))
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(data)
	return nil
}

// writeJSONError logs the error and writes an apiError with the given HTTP status code: server errors (5xx) report only the generic status text so internal details are not leaked, client errors (4xx) keep the original message.
func writeJSONError(w http.ResponseWriter, status int, err error) {
	var message string
	if status >= http.StatusInternalServerError {
		Logger.Error("request error", zap.Int("status", status), zap.Error(err))
		message = http.StatusText(status)
	} else {
		Logger.Debug("request error", zap.Int("status", status), zap.Error(err))
		message = err.Error()
	}

	if writeErr := writeJSON(w, status, apiError{Code: status, Message: message}); writeErr != nil {
		Logger.Error("failed to write json", zap.Error(writeErr))
	}
}

// decodeJSON unmarshals the request body into target and classifies decode errors.
func decodeJSON(r *http.Request, target any) error {
	var syntaxErr *json.SyntaxError
	var unmarshalTypeErr *json.UnmarshalTypeError

	err := json.NewDecoder(r.Body).Decode(target)

	if errors.As(err, &syntaxErr) {
		return &decodeError{
			status:  http.StatusBadRequest,
			message: fmt.Sprintf("invalid JSON syntax at offset %d", syntaxErr.Offset),
			err:     err,
		}
	}

	if errors.As(err, &unmarshalTypeErr) {
		return &decodeError{
			status:  http.StatusBadRequest,
			message: fmt.Sprintf("invalid type %q for field %q", unmarshalTypeErr.Value, unmarshalTypeErr.Field),
			err:     err,
		}
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &decodeError{
			status:  http.StatusBadRequest,
			message: "request body is empty",
			err:     err,
		}
	}

	if err != nil {
		return &decodeError{
			status:  http.StatusInternalServerError,
			message: "failed to decode request body",
			err:     err,
		}
	}
	return nil
}

// decodeErrorStatus extracts the HTTP status code from a decodeError, defaulting to 500.
func decodeErrorStatus(err error) int {
	var decodeErr *decodeError
	if errors.As(err, &decodeErr) {
		return decodeErr.status
	}
	return http.StatusInternalServerError
}

// handleJSONUpdate handles POST /update: validates the JSON payload, stores the metric,
// and returns the updated metric value in the response body.
func (h *MetricsHandler) handleJSONUpdate(w http.ResponseWriter, r *http.Request) {
	var metric model.Metric

	if err := decodeJSON(r, &metric); err != nil {
		writeJSONError(w, decodeErrorStatus(err), err)
		return
	}

	if err := metric.ValidateForUpdate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	switch metric.Type {
	case model.Gauge:
		if err := h.storage.SetGauge(metric.ID, *metric.Value); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
	case model.Counter:
		newDelta, err := h.storage.AddCounter(metric.ID, *metric.Delta)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		metric.Delta = &newDelta
	}
	if err := writeJSON(w, http.StatusOK, metric); err != nil {
		Logger.Error("failed to write json", zap.Error(err))
		return
	}
}

// handleJSONUpdates handles POST /updates: validates and stores a batch of metrics.
func (h *MetricsHandler) handleJSONUpdates(w http.ResponseWriter, r *http.Request) {
	var metrics []model.Metric

	if err := decodeJSON(r, &metrics); err != nil {
		writeJSONError(w, decodeErrorStatus(err), err)
		return
	}

	for _, metric := range metrics {
		if err := metric.ValidateForUpdate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, err)
			return
		}
	}

	err := h.storage.UpdateMetrics(r.Context(), metrics)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleJSONRead handles POST /value: validates the JSON payload, looks up the stored
// metric and returns its current value in the response body.
func (h *MetricsHandler) handleJSONRead(w http.ResponseWriter, r *http.Request) {
	var metric model.Metric

	if err := decodeJSON(r, &metric); err != nil {
		writeJSONError(w, decodeErrorStatus(err), err)
		return
	}

	if err := metric.ValidateForRead(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	switch metric.Type {
	case model.Gauge:
		value, err := h.storage.GetGauge(metric.ID)
		if errors.Is(err, model.ErrMetricNotFound) {
			writeJSONError(w, http.StatusNotFound, err)
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		metric.Value = &value

	case model.Counter:
		delta, err := h.storage.GetCounter(metric.ID)
		if errors.Is(err, model.ErrMetricNotFound) {
			writeJSONError(w, http.StatusNotFound, err)
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		metric.Delta = &delta
	}
	if err := writeJSON(w, http.StatusOK, metric); err != nil {
		Logger.Error("failed to write json", zap.Error(err))
		return
	}
}
