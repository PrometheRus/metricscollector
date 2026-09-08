package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/nikitaw13/metricscollector/internal/model"
)

// Sender is responsible for sending collected metrics to the server
// as a single batched HTTP POST request.
type Sender struct {
	baseURL string
	storage Storage
	client  HTTPClient
}

// NewSender creates a Sender with the given base URL, storage, and HTTP client.
func NewSender(baseURL string, storage Storage, client HTTPClient) *Sender {
	return &Sender{
		baseURL: baseURL,
		storage: storage,
		client:  client,
	}
}

// Run performs a one-shot send of all stored gauge and counter metrics to the server.
// Counters are drained into the batch before sending; if delivery fails, their
// drained values are merged back into storage for the next attempt.
func (s *Sender) Run() {
	updatesURL := fmt.Sprintf("%s/updates", s.baseURL)
	var metrics []model.Metric

	for metricName, value := range s.storage.GetAllGauges() {
		metric := model.Metric{
			Type:  model.Gauge,
			ID:    metricName,
			Value: &value,
		}
		metrics = append(metrics, metric)
	}

	drained := s.storage.DrainCounters()
	for metricName, value := range drained {
		metric := model.Metric{
			Type:  model.Counter,
			ID:    metricName,
			Delta: &value,
		}
		metrics = append(metrics, metric)
	}

	// If delivery fails, merge the drained counter values back into storage.
	committed := false
	defer func() {
		if !committed {
			restoreCounters(s.storage, drained)
		}
	}()

	if len(metrics) == 0 {
		log.Println("no metrics to send")
		return
	}

	jsonBody, err := json.Marshal(&metrics)
	if err != nil {
		log.Println(err)
		return
	}

	compressedBody, err := Compress(jsonBody)
	if err != nil {
		log.Println(err)
		return
	}
	req, err := http.NewRequest(
		http.MethodPost,
		updatesURL,
		bytes.NewReader(compressedBody),
	)
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Content-Encoding", "gzip")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}

	logRequest(resp, updatesURL)
	drainAndCloseResponse(resp)

	if resp.StatusCode >= http.StatusMultipleChoices {
		return
	}
	committed = true
}

// restoreCounters merges drained counter values back into storage after a
// failed send; the additive merge preserves increments collected in flight.
func restoreCounters(storage Storage, drained map[string]int64) {
	for name, value := range drained {
		storage.AddCounter(name, value)
	}
}

// logRequest logs the outcome of sending a metrics batch.
func logRequest(resp *http.Response, url string) {
	if resp.StatusCode < 300 {
		log.Printf("Metrics sent to %s", url)
	} else {
		log.Printf("Metrics failed to send to %s, status: %d", url, resp.StatusCode)
	}
}

// drainAndCloseResponse drains and closes the response body to allow TCP connection reuse.
func drainAndCloseResponse(resp *http.Response) {
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
