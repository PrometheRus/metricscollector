package repository

import (
	"context"
	"fmt"
	"log"
	"maps"
	"sync"

	"github.com/nikitaw13/metricscollector/internal/model"
)

// MemStorage implements in-memory metric storage using separate maps for gauge and counter metrics.
type MemStorage struct {
	mu      sync.RWMutex
	gauge   map[string]float64
	counter map[string]int64
}

// NewMemStorage creates and returns an initialized MemStorage with empty metric maps.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   map[string]float64{},
		counter: map[string]int64{},
	}
}

// SetGauge sets the named gauge metric to the specified value, overwriting any previous value.
func (ms *MemStorage) SetGauge(name string, value float64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.gauge[name] = value
	return nil
}

// AddCounter increments the named counter metric by the specified delta.
func (ms *MemStorage) AddCounter(name string, delta int64) (newDelta int64, err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.counter[name] += delta
	return ms.counter[name], nil
}

// GetGauge returns the value of the named gauge metric.
// Returns an error if the metric does not exist.
func (ms *MemStorage) GetGauge(name string) (float64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if _, ok := ms.gauge[name]; ok {
		return ms.gauge[name], nil
	}
	return 0, fmt.Errorf("gauge %s %w", name, model.ErrMetricNotFound)
}

// GetCounter returns the value of the named counter metric.
// Returns an error if the metric does not exist.
func (ms *MemStorage) GetCounter(name string) (int64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if _, ok := ms.counter[name]; ok {
		return ms.counter[name], nil
	}
	return 0, fmt.Errorf("counter %s %w", name, model.ErrMetricNotFound)
}

// GetAllGauges returns a shallow copy of all gauge metrics to prevent external mutation.
func (ms *MemStorage) GetAllGauges() (map[string]float64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return maps.Clone(ms.gauge), nil
}

// GetAllCounters returns a shallow copy of all counter metrics to prevent external mutation.
func (ms *MemStorage) GetAllCounters() (map[string]int64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return maps.Clone(ms.counter), nil
}

// UpdateMetrics applies a batch of metric updates in a single lock acquisition.
func (ms *MemStorage) UpdateMetrics(ctx context.Context, metrics []model.Metric) (err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, metric := range metrics {
		switch metric.Type {
		case model.Counter:
			ms.counter[metric.ID] += *metric.Delta
		case model.Gauge:
			ms.gauge[metric.ID] = *metric.Value
		default:
			log.Printf("unknown metric type: %s\n", metric.Type)
		}
	}

	return nil
}

// LoadMetrics replaces all stored metrics with the provided snapshot.
func (ms *MemStorage) LoadMetrics(metrics []model.Metric) (err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	clear(ms.gauge)
	clear(ms.counter)

	for _, metric := range metrics {
		switch metric.Type {
		case model.Counter:
			ms.counter[metric.ID] = *metric.Delta
		case model.Gauge:
			ms.gauge[metric.ID] = *metric.Value
		default:
			log.Printf("unknown metric type: %s\n", metric.Type)
		}

	}

	return nil
}

// PingContext implements DBPinger for in-memory storage: there is no
// external database to check, so the ping always succeeds.
func (ms *MemStorage) PingContext(ctx context.Context) error {
	return nil
}
