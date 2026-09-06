package agent

import "testing"

// TestAllMetricsExist verifies that collector populates all required gauge and counter metrics.
func TestAllMetricsExist(t *testing.T) {
	storage := NewAgentStorage()
	collector := NewCollector(storage)
	collector.Run()
	for _, metric := range GaugeMetrics {
		_, err := storage.GetGauge(metric)
		if err != nil {
			t.Errorf("Gauge metric %q not found", metric)
		}
	}
	for _, metric := range CounterMetrics {
		_, err := storage.GetCounter(metric)
		if err != nil {
			t.Errorf("Counter metric %q not found", metric)
		}
	}
}

// TestPollCountIncrement verifies that PollCount increases by 1 on each collector run.
func TestPollCountIncrement(t *testing.T) {
	storage := NewAgentStorage()
	collector := NewCollector(storage)

	for i := 1; i < 100; i++ {
		collector.Run()
		pollCount, _ := storage.GetCounter("PollCount")
		if pollCount != int64(i) {
			t.Errorf("PollCount = %d, want %d", pollCount, i)
		}
	}
}
