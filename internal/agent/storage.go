package agent

import (
	"fmt"
	"maps"
	"sync"

	"github.com/nikitaw13/metricscollector/internal/model"
)

// MetricReader defines read-only operations for retrieving metrics.
type MetricReader interface {
	GetAllGauges() map[string]float64
	DrainCounters() map[string]int64
}

// MetricWriter defines write operations for updating metrics.
type MetricWriter interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
}

// Storage combines read and write operations for the agent's metric store.
type Storage interface {
	MetricReader
	MetricWriter
}

// AgentStorage is an in-memory implementation of Storage that holds agent metrics.
type AgentStorage struct {
	mu      sync.Mutex
	gauge   map[string]float64
	counter map[string]int64
}

// Compile-time check that AgentStorage implements Storage.
var _ Storage = (*AgentStorage)(nil)

// NewAgentStorage creates and returns a new AgentStorage with initialized maps.
func NewAgentStorage() *AgentStorage {
	return &AgentStorage{
		gauge:   map[string]float64{},
		counter: map[string]int64{},
	}
}

// SetGauge sets the gauge metric with the given name to the specified value.
func (ms *AgentStorage) SetGauge(name string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.gauge[name] = value
}

// AddCounter increments the counter metric with the given name by the specified value.
func (ms *AgentStorage) AddCounter(name string, value int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.counter[name] += value
}

// GetGauge returns the value of the gauge metric by name.
// Returns an error if the metric does not exist.
func (ms *AgentStorage) GetGauge(name string) (float64, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, ok := ms.gauge[name]; ok {
		return ms.gauge[name], nil
	}
	return 0, fmt.Errorf("gauge %s %w", name, model.ErrMetricNotFound)
}

// GetCounter returns the value of the counter metric by name.
// Returns an error if the metric does not exist.
func (ms *AgentStorage) GetCounter(name string) (int64, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, ok := ms.counter[name]; ok {
		return ms.counter[name], nil
	}
	return 0, fmt.Errorf("counter %s %w", name, model.ErrMetricNotFound)
}

// GetAllGauges returns a copy of all gauge metrics to prevent external mutation.
func (ms *AgentStorage) GetAllGauges() map[string]float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return maps.Clone(ms.gauge)
}

// GetAllCounters returns a copy of all counter metrics to prevent external mutation.
func (ms *AgentStorage) GetAllCounters() map[string]int64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return maps.Clone(ms.counter)
}

// DrainCounters atomically returns current counter values and resets them to zero.
func (ms *AgentStorage) DrainCounters() map[string]int64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	snapshot := maps.Clone(ms.counter)
	clear(ms.counter)
	return snapshot
}
