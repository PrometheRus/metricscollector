package repository

import (
	"reflect"
	"sync"
	"testing"

	"github.com/nikitaw13/metricscollector/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMemStorage verifies that the constructor returns an initialized empty storage.
func TestNewMemStorage(t *testing.T) {
	tests := []struct {
		name string
		want *MemStorage
	}{
		{
			name: "Return NewMemStorage instance",
			want: &MemStorage{
				gauge:   map[string]float64{},
				counter: map[string]int64{},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewMemStorage(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("NewMemStorage() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestSetGauge verifies that SetGauge accepts values of any sign without error.
func TestSetGauge(t *testing.T) {
	type fields struct {
		gauge   map[string]float64
		counter map[string]int64
	}
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "gauge value is 1.00",
			fields: fields{
				gauge:   map[string]float64{"metric_gauge": 1.00},
				counter: map[string]int64{"metric_counter": 1},
			},
			args:    args{name: "metric_gauge", value: 1.00},
			wantErr: false,
		},
		{
			name: "gauge value is -1.00",
			fields: fields{
				gauge:   map[string]float64{"metric_gauge": -1.00},
				counter: map[string]int64{"metric_counter": 1},
			},
			args:    args{name: "metric_gauge", value: -1.00},
			wantErr: false,
		},
		{
			name: "gauge value is 1111.50",
			fields: fields{
				gauge:   map[string]float64{"metric_gauge": 1111.50},
				counter: map[string]int64{"metric_counter": 1},
			},
			args:    args{name: "metric_gauge", value: 1111.50},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ms := &MemStorage{
				gauge:   tc.fields.gauge,
				counter: tc.fields.counter,
			}
			if err := ms.SetGauge(tc.args.name, tc.args.value); (err != nil) != tc.wantErr {
				t.Errorf("MemStorage.SetGauge() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestAddCounter verifies that deltas accumulate and the new total is returned.
func TestAddCounter(t *testing.T) {
	type fields struct {
		gauge   map[string]float64
		counter map[string]int64
	}
	type args struct {
		name  string
		delta int64
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantDelta int64 // accumulated value expected after AddCounter
		wantErr   bool
	}{
		{
			name: "counter value is 1",
			fields: fields{
				gauge:   map[string]float64{"metric_gauge": 1.00},
				counter: map[string]int64{"metric_counter": 1},
			},
			args:      args{name: "metric_counter", delta: 1},
			wantDelta: 2,
			wantErr:   false,
		},
		{
			name: "counter value is -11111",
			fields: fields{
				gauge:   map[string]float64{"metric_gauge": 1.00},
				counter: map[string]int64{"metric_counter": -1111},
			},
			args:      args{name: "metric_counter", delta: -1111},
			wantDelta: -2222,
			wantErr:   false,
		},
		{
			name: "counter value is 1111",
			fields: fields{
				gauge:   map[string]float64{"metric_gauge": 1.00},
				counter: map[string]int64{"metric_counter": 1},
			},
			args:      args{name: "metric_counter", delta: 1111},
			wantDelta: 1112,
			wantErr:   false,
		},
		{
			name: "counter value is 0",
			fields: fields{
				gauge:   map[string]float64{"metric_gauge": 1.00},
				counter: map[string]int64{"metric_counter": 1},
			},
			args:      args{name: "metric_counter", delta: 0},
			wantDelta: 1,
			wantErr:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ms := &MemStorage{
				gauge:   tc.fields.gauge,
				counter: tc.fields.counter,
			}

			newDelta, err := ms.AddCounter(tc.args.name, tc.args.delta)

			if (err != nil) != tc.wantErr {
				t.Errorf("MemStorage.AddCounter() error = %v, wantErr %v", err, tc.wantErr)
			}
			if newDelta != tc.wantDelta {
				t.Errorf("MemStorage.AddCounter() newDelta = %v, want %v", newDelta, tc.wantDelta)
			}
		})
	}
}

// TestUpdateMetrics verifies that a batch of metric updates is applied:
// gauges overwrite previous values, counters accumulate, and metrics with
// an unknown type are skipped without creating entries.
func TestUpdateMetrics(t *testing.T) {
	t.Parallel()

	gaugeVal := 42.5
	counterVal := int64(7)
	metrics := []model.Metric{
		{ID: "batch_gauge", Type: model.Gauge, Value: &gaugeVal},
		{ID: "batch_counter", Type: model.Counter, Delta: &counterVal},
		{ID: "batch_unknown", Type: "random"},
	}

	ms := NewMemStorage()
	require.NoError(t, ms.SetGauge("batch_gauge", 1.0))
	newDelta, err := ms.AddCounter("batch_counter", 3)
	require.NoError(t, err)
	require.Equal(t, int64(3), newDelta)

	require.NoError(t, ms.UpdateMetrics(t.Context(), metrics))

	gotGauge, err := ms.GetGauge("batch_gauge")
	require.NoError(t, err)
	assert.InDelta(t, 42.5, gotGauge, 0.001, "gauge must be overwritten by the batch")

	gotCounter, err := ms.GetCounter("batch_counter")
	require.NoError(t, err)
	assert.Equal(t, int64(10), gotCounter, "counter must accumulate through the batch")

	_, err = ms.GetGauge("batch_unknown")
	assert.ErrorIs(t, err, model.ErrMetricNotFound, "unknown metric type must not create entries")
}

// TestUpdateMetrics_Concurrent verifies that concurrent batch and single
// updates do not race and that all counter deltas are accounted for.
// Run with -race to catch data races.
func TestUpdateMetrics_Concurrent(t *testing.T) {
	t.Parallel()

	ms := NewMemStorage()

	const goroutines = 10
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			delta := int64(1)
			_ = ms.UpdateMetrics(t.Context(), []model.Metric{
				{ID: "concurrent_counter", Type: model.Counter, Delta: &delta},
			})
		}()
		go func() {
			defer wg.Done()
			_ = ms.SetGauge("concurrent_gauge", 1.0)
		}()
	}
	wg.Wait()

	gotCounter, err := ms.GetCounter("concurrent_counter")
	require.NoError(t, err)
	assert.Equal(t, int64(goroutines), gotCounter, "all concurrent deltas must be applied")

	gotGauge, err := ms.GetGauge("concurrent_gauge")
	require.NoError(t, err)
	assert.InDelta(t, 1.0, gotGauge, 0.001)
}
