package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nikitaw13/metricscollector/internal/agent"
)

func main() {
	parseFlags()
	parseEnvs()
	run()
}

func run() {
	var (
		baseURL     = fmt.Sprintf("http://%s", flagServerAddr)
		timeouts    = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
		httpClient  = &http.Client{Timeout: 5 * time.Second}
		retryClient = agent.NewClientWithRetries(timeouts, httpClient)
		storage     = agent.NewAgentStorage()
		sender      = agent.NewSender(baseURL, storage, retryClient)
		collector   = agent.NewCollector(storage)
	)

	// Collector runs in a separate goroutine since two independent intervals
	// cannot be managed by Sleep in a single goroutine.
	go func() {
		for {
			collector.Run()
			time.Sleep(time.Duration(flagPollInterval) * time.Second)
		}
	}()

	for {
		sender.Run()
		time.Sleep(time.Duration(flagReportInterval) * time.Second)
	}
}
