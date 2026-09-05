package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/nikitaw13/metricscollector/internal/model"
)

// PersistentMemStorage wraps MemStorage with file-based persistence capabilities.
type PersistentMemStorage struct {
	*MemStorage
	filePath  string
	syncWrite bool
}

// NewPersistentMemStorage creates a PersistentMemStorage with the given underlying storage, file path, and sync-write mode.
func NewPersistentMemStorage(memStorage *MemStorage, filePath string, syncWrite bool) *PersistentMemStorage {
	return &PersistentMemStorage{
		MemStorage: memStorage,
		filePath:   filePath,
		syncWrite:  syncWrite,
	}
}

// Restore loads previously saved metrics from the file into in-memory maps.
func (ps *PersistentMemStorage) Restore() error {
	file, err := os.Open(ps.filePath)

	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	var metrics []model.Metric
	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return fmt.Errorf("error unmarshaling metrics %v, %w", data, err)
	}

	err = ps.MemStorage.LoadMetrics(metrics)
	if err != nil {
		return fmt.Errorf("error restoring gauge: %w", err)
	}

	return nil

}

// Save writes all current metrics to the file, replacing any previous contents.
func (ps *PersistentMemStorage) Save() error {
	file, err := os.OpenFile(ps.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	defer file.Close()

	gauges, _ := ps.GetAllGauges()
	counters, _ := ps.GetAllCounters()
	var metrics []model.Metric

	for name, value := range gauges {
		metric := model.Metric{ID: name, Type: "gauge", Value: &value}
		metrics = append(metrics, metric)
	}

	for name, value := range counters {
		metric := model.Metric{ID: name, Type: "counter", Delta: &value}
		metrics = append(metrics, metric)
	}

	data, err := json.Marshal(&metrics)
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

// PeriodicSave saves metrics to disk at the given interval until the program exits.
func (ps *PersistentMemStorage) PeriodicSave(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		err := ps.Save()
		if err != nil {
			log.Printf("error writing file: %v", err)
			continue
		}
	}
}

// SaveSync performs a synchronous save and logs any error.
func (ps *PersistentMemStorage) SaveSync() {
	err := ps.Save()
	if err != nil {
		log.Printf("error writing file: %v", err)
	}
}

// SetGauge sets the named gauge metric to the specified value, overwriting any previous value.
func (ps *PersistentMemStorage) SetGauge(name string, value float64) (err error) {
	err = ps.MemStorage.SetGauge(name, value)
	if err != nil {
		return fmt.Errorf("error SetGauge: %w", err)
	}

	if ps.syncWrite {
		ps.SaveSync()
	}
	return nil
}

// AddCounter increments the named counter metric by the specified delta.
func (ps *PersistentMemStorage) AddCounter(name string, delta int64) (newDelta int64, err error) {
	newDelta, err = ps.MemStorage.AddCounter(name, delta)
	if err != nil {
		return 0, fmt.Errorf("error AddCounter: %w", err)
	}

	if ps.syncWrite {
		ps.SaveSync()
	}
	return newDelta, nil
}

// UpdateMetrics applies a batch of metric updates and saves to disk when sync-write is enabled.
func (ps *PersistentMemStorage) UpdateMetrics(ctx context.Context, metrics []model.Metric) (err error) {
	err = ps.MemStorage.UpdateMetrics(ctx, metrics)
	if err != nil {
		return fmt.Errorf("error UpdateMetrics: %w", err)
	}

	if ps.syncWrite {
		ps.SaveSync()
	}

	return nil
}
