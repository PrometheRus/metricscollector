package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ptrFloat64 returns a pointer to value, used to build *float64 for test expectations.
func ptrFloat64(value float64) *float64 { return &value }

// ptrInt64 returns a pointer to value, used to build *int64 for test expectations.
func ptrInt64(value int64) *int64 { return &value }

// TestMetric_ValidateForUpdate verifies that invalid type, ID, or value are rejected and valid metrics pass.
func TestMetric_ValidateForUpdate(t *testing.T) {
	type fields struct {
		ID    string
		Type  string
		Delta *int64
		Value *float64
	}
	tests := []struct {
		name          string
		fields        fields
		expectedError error
	}{
		// Negative tests
		{
			name:          "ErrMissingType",
			expectedError: ErrMissingType,
			fields: fields{
				ID:    "",
				Type:  "",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "ErrInvalidType",
			expectedError: ErrInvalidType,
			fields: fields{
				ID:    "",
				Type:  "random",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "ErrMissingID",
			expectedError: ErrMissingID,
			fields: fields{
				ID:    "",
				Type:  "counter",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "ErrMissingValue for Gauge",
			expectedError: ErrMissingValue,
			fields: fields{
				ID:    "metric",
				Type:  "gauge",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "ErrMissingValue for Counter",
			expectedError: ErrMissingValue,
			fields: fields{
				ID:    "metric",
				Type:  "counter",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "ErrInvalidValue for +infinity",
			expectedError: ErrInvalidValue,
			fields: fields{
				ID:    "metric",
				Type:  "gauge",
				Delta: nil,
				Value: ptrFloat64(math.Inf(1)),
			},
		},
		{
			name:          "ErrInvalidValue for NaN",
			expectedError: ErrInvalidValue,
			fields: fields{
				ID:    "metric",
				Type:  "gauge",
				Delta: nil,
				Value: ptrFloat64(math.NaN()),
			},
		},
		{
			name:          "ErrInvalidValue",
			expectedError: ErrInvalidValue,
			fields: fields{
				ID:    "metric",
				Type:  "counter",
				Delta: ptrInt64(-1),
				Value: nil,
			},
		},
		// Positive tests
		{
			name:          "Valid counter request",
			expectedError: nil,
			fields: fields{
				ID:    "metric",
				Type:  "counter",
				Delta: ptrInt64(1),
				Value: nil,
			},
		},
		{
			name:          "Valid gauge request",
			expectedError: nil,
			fields: fields{
				ID:    "metric",
				Type:  "gauge",
				Delta: nil,
				Value: ptrFloat64(1.00),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metric := Metric{
				ID:    tc.fields.ID,
				Type:  tc.fields.Type,
				Delta: tc.fields.Delta,
				Value: tc.fields.Value,
			}

			err := metric.ValidateForUpdate()
			assert.ErrorIs(t, err, tc.expectedError, "error = %v, expectedError %v", err, tc.expectedError)
		})
	}
}

// TestMetric_ValidateForRead verifies that only a valid type and ID are required for reads.
func TestMetric_ValidateForRead(t *testing.T) {
	type fields struct {
		ID    string
		Type  string
		Delta *int64
		Value *float64
	}
	tests := []struct {
		name          string
		fields        fields
		expectedError error
	}{
		// Negative tests
		{
			name:          "ErrMissingType",
			expectedError: ErrMissingType,
			fields: fields{
				ID:    "",
				Type:  "",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "ErrInvalidType",
			expectedError: ErrInvalidType,
			fields: fields{
				ID:    "",
				Type:  "random",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "ErrMissingID",
			expectedError: ErrMissingID,
			fields: fields{
				ID:    "",
				Type:  "counter",
				Delta: nil,
				Value: nil,
			},
		},
		// Positive tests
		{
			name:          "Valid counter request",
			expectedError: nil,
			fields: fields{
				ID:    "metric",
				Type:  "counter",
				Delta: nil,
				Value: nil,
			},
		},
		{
			name:          "Valid gauge request",
			expectedError: nil,
			fields: fields{
				ID:    "metric",
				Type:  "gauge",
				Delta: nil,
				Value: nil,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metric := Metric{
				ID:    tc.fields.ID,
				Type:  tc.fields.Type,
				Delta: tc.fields.Delta,
				Value: tc.fields.Value,
			}

			err := metric.ValidateForRead()
			assert.ErrorIs(t, err, tc.expectedError, "error = %v, expectedError %v", err, tc.expectedError)
		})
	}
}
