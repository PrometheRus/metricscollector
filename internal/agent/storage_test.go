package agent

import (
	"math/rand/v2"
	"testing"

	"github.com/nikitaw13/metricscollector/internal/model"
	"github.com/stretchr/testify/require"
)

// TestSetGauge verifies that SetGauge stores the value and overwrites on subsequent calls.
func TestSetGauge(t *testing.T) {
	ms := NewAgentStorage()
	for _, metricName := range GaugeMetrics {
		t.Run(metricName, func(t *testing.T) {
			firstRandomValue := rand.Float64()
			ms.SetGauge(metricName, firstRandomValue)
			firstGet, firstErr := ms.GetGauge(metricName)
			require.NoError(t, firstErr)
			require.Equal(t, firstRandomValue, firstGet)

			secondRandomValue := rand.Float64()
			ms.SetGauge(metricName, secondRandomValue)
			secondGet, secondErr := ms.GetGauge(metricName)
			require.NoError(t, secondErr)
			require.Equal(t, secondRandomValue, secondGet)
		})
	}
}

// TestAddCounter verifies that AddCounter increments the value cumulatively.
func TestAddCounter(t *testing.T) {
	ms := NewAgentStorage()
	for _, metricName := range CounterMetrics {
		t.Run(metricName, func(t *testing.T) {
			firstRandomValue := rand.Int64N(10)
			ms.AddCounter(metricName, firstRandomValue)
			firstGet, firstErr := ms.GetCounter(metricName)
			require.NoError(t, firstErr)
			require.Equal(t, firstRandomValue, firstGet)

			secondRandomValue := rand.Int64N(10)
			ms.AddCounter(metricName, secondRandomValue)
			secondGet, secondErr := ms.GetCounter(metricName)
			require.NoError(t, secondErr)
			require.Equal(t, firstRandomValue+secondRandomValue, secondGet)
		})
	}
}

// TestGetGaugeError verifies that GetGauge returns an error for a nonexistent key.
func TestGetGaugeError(t *testing.T) {
	ms := NewAgentStorage()
	t.Run("nonexistent gauge", func(t *testing.T) {
		_, err := ms.GetGauge("__nonexistent__")
		require.ErrorIs(t, err, model.ErrMetricNotFound)
	})
}

// TestGetCounterError verifies that GetCounter returns an error for a nonexistent key.
func TestGetCounterError(t *testing.T) {
	ms := NewAgentStorage()
	t.Run("nonexistent counter", func(t *testing.T) {
		_, err := ms.GetCounter("__nonexistent__")
		require.ErrorIs(t, err, model.ErrMetricNotFound)
	})
}

// TestGetAllGaugesReturnsCopy verifies that GetAllGauges returns a copy, not a reference.
func TestGetAllGaugesReturnsCopy(t *testing.T) {
	ms := NewAgentStorage()
	gaugeCopy := ms.GetAllGauges()

	for _, metricName := range GaugeMetrics {
		t.Run(metricName, func(t *testing.T) {
			firstRandomValue := rand.Float64()
			ms.SetGauge(metricName, firstRandomValue)
			require.NotEqual(t, gaugeCopy[metricName], firstRandomValue)

			secondRandomValue := rand.Float64()
			ms.SetGauge(metricName, secondRandomValue)
			require.NotEqual(t, gaugeCopy[metricName], secondRandomValue)
		})
	}
}

// TestGetAllCountersReturnsCopy verifies that GetAllCounters returns a copy, not a reference.
func TestGetAllCountersReturnsCopy(t *testing.T) {
	ms := NewAgentStorage()
	counterCopy := ms.GetAllCounters()

	for _, metricName := range CounterMetrics {
		t.Run(metricName, func(t *testing.T) {
			firstRandomValue := rand.Int64()
			ms.AddCounter(metricName, firstRandomValue)
			require.NotEqual(t, counterCopy[metricName], firstRandomValue)

			secondRandomValue := rand.Int64()
			ms.AddCounter(metricName, secondRandomValue)
			require.NotEqual(t, counterCopy[metricName], secondRandomValue)
		})
	}
}

// TestGetAllGauges verifies that GetAllGauges returns a map containing all stored values.
func TestGetAllGauges(t *testing.T) {
	ms := NewAgentStorage()
	for _, metricName := range GaugeMetrics {
		t.Run(metricName, func(t *testing.T) {
			initialValue := rand.Float64()
			ms.SetGauge(metricName, initialValue)
			got, err := ms.GetGauge(metricName)

			gauges := ms.GetAllGauges()
			require.NoError(t, err)
			require.Equal(t, gauges[metricName], got)
		})

	}
}

// TestGetAllCounters verifies that GetAllCounters returns a map containing all stored values.
func TestGetAllCounters(t *testing.T) {
	ms := NewAgentStorage()
	for _, metricName := range CounterMetrics {
		t.Run(metricName, func(t *testing.T) {
			initialValue := rand.Int64()
			ms.AddCounter(metricName, initialValue)
			got, err := ms.GetCounter(metricName)

			counters := ms.GetAllCounters()
			require.NoError(t, err)
			require.Equal(t, counters[metricName], got)
		})
	}
}
