package model

import (
	"errors"
	"math"
)

// Supported metric types.
const (
	Counter = "counter"
	Gauge   = "gauge"
)

// Errors describing invalid metric payloads and missing metrics.
var (
	ErrMissingType    = errors.New("Metric type is required")
	ErrInvalidType    = errors.New("invalid metric type")
	ErrMissingID      = errors.New("Metric ID is required")
	ErrMissingValue   = errors.New("Metric value is required")
	ErrInvalidValue   = errors.New("invalid metric value")
	ErrMetricNotFound = errors.New("not found")
)

// Metric represents a flat metric model.
// Delta and Value are pointers to distinguish 0 from an unset value.
// Type must be one of Counter or Gauge.
type Metric struct {
	ID    string   `json:"id"`
	Type  string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"` // increment for counter type
	Value *float64 `json:"value,omitempty"` // absolute value for gauge type
	Hash  string   `json:"hash,omitempty"`
}

// ValidateForUpdate checks that the metric has a valid type, ID and value for an update request.
func (m Metric) ValidateForUpdate() error {
	if m.Type == "" {
		return ErrMissingType
	}
	if m.Type != Gauge && m.Type != Counter {
		return ErrInvalidType
	}
	if m.ID == "" {
		return ErrMissingID
	}

	if (m.Type == Gauge && m.Value == nil) || (m.Type == Counter && m.Delta == nil) {
		return ErrMissingValue
	}

	if m.Type == Gauge && m.Value != nil && math.IsInf(*m.Value, 0) {
		return ErrInvalidValue
	}
	if m.Type == Gauge && m.Value != nil && math.IsNaN(*m.Value) {
		return ErrInvalidValue
	}

	if m.Type == Counter && m.Delta != nil && *m.Delta < 0 {
		return ErrInvalidValue
	}

	return nil
}

// ValidateForRead checks that the metric has a valid type and ID for a read request.
func (m Metric) ValidateForRead() error {
	if m.Type == "" {
		return ErrMissingType
	}
	if m.Type != Gauge && m.Type != Counter {
		return ErrInvalidType
	}
	if m.ID == "" {
		return ErrMissingID
	}
	return nil
}
