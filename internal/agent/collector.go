package agent

import (
	"log"
	"math/rand/v2"
	"runtime"
)

// Collector gathers system metrics and stores them in the provided Storage.
type Collector struct {
	storage Storage
}

// NewCollector creates a Collector that stores collected metrics in the given storage.
func NewCollector(storage Storage) *Collector {
	return &Collector{
		storage: storage,
	}
}

// Run collects one snapshot of runtime memory statistics and custom metrics.
func (c *Collector) Run() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.storage.SetGauge("Alloc", float64(memStats.Alloc))
	c.storage.SetGauge("BuckHashSys", float64(memStats.BuckHashSys))
	c.storage.SetGauge("Frees", float64(memStats.Frees))
	c.storage.SetGauge("GCCPUFraction", float64(memStats.GCCPUFraction))
	c.storage.SetGauge("GCSys", float64(memStats.GCSys))
	c.storage.SetGauge("HeapAlloc", float64(memStats.HeapAlloc))
	c.storage.SetGauge("HeapIdle", float64(memStats.HeapIdle))
	c.storage.SetGauge("HeapInuse", float64(memStats.HeapInuse))
	c.storage.SetGauge("HeapObjects", float64(memStats.HeapObjects))
	c.storage.SetGauge("HeapReleased", float64(memStats.HeapReleased))
	c.storage.SetGauge("HeapSys", float64(memStats.HeapSys))
	c.storage.SetGauge("LastGC", float64(memStats.LastGC))
	c.storage.SetGauge("Lookups", float64(memStats.Lookups))
	c.storage.SetGauge("MCacheInuse", float64(memStats.MCacheInuse))
	c.storage.SetGauge("MCacheSys", float64(memStats.MCacheSys))
	c.storage.SetGauge("MSpanInuse", float64(memStats.MSpanInuse))
	c.storage.SetGauge("MSpanSys", float64(memStats.MSpanSys))
	c.storage.SetGauge("Mallocs", float64(memStats.Mallocs))
	c.storage.SetGauge("NextGC", float64(memStats.NextGC))
	c.storage.SetGauge("NumForcedGC", float64(memStats.NumForcedGC))
	c.storage.SetGauge("NumGC", float64(memStats.NumGC))
	c.storage.SetGauge("OtherSys", float64(memStats.OtherSys))
	c.storage.SetGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	c.storage.SetGauge("StackInuse", float64(memStats.StackInuse))
	c.storage.SetGauge("StackSys", float64(memStats.StackSys))
	c.storage.SetGauge("Sys", float64(memStats.Sys))
	c.storage.SetGauge("TotalAlloc", float64(memStats.TotalAlloc))
	// Custom Metrics
	c.storage.AddCounter("PollCount", 1)                  // increments by 1 on each collection cycle
	c.storage.SetGauge("RandomValue", rand.NormFloat64()) // random normally-distributed value
	log.Print("metrics collected")
}
