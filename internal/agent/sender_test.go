package agent

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitaw13/metricscollector/internal/model"
	"github.com/stretchr/testify/assert"
)

// expectedContentType is the expected Content-Type header for all JSON requests from the agent.
const expectedContentType = "application/json; charset=utf-8"

// TestSendMetrics verifies that the sender reports all gauge and counter
// metrics to the server in a single batched request. It checks:
//   - exactly one request is made per Run(),
//   - each metric was received by the server,
//   - the HTTP method is POST,
//   - the Content-Type header is "application/json; charset=utf-8".
func TestSendMetrics(t *testing.T) {
	storage := NewAgentStorage()

	for _, metricName := range GaugeMetrics {
		initialValue := rand.Float64()
		storage.SetGauge(metricName, initialValue)
	}

	for _, metricName := range CounterMetrics {
		initialValue := rand.Int64()
		storage.AddCounter(metricName, initialValue)
	}

	var mu sync.Mutex
	received := map[string]bool{}
	contentType := map[string]string{}
	method := map[string]string{}
	requestCount := 0

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		var metrics []model.Metric
		if err := json.NewDecoder(gz).Decode(&metrics); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		requestCount++
		for _, metric := range metrics {
			received[metric.ID] = true
			method[metric.ID] = r.Method
			contentType[metric.ID] = r.Header.Get("Content-Type")
		}
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(testHandler)
	defer ts.Close()

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second},
	)

	sender := NewSender(
		ts.URL,
		storage,
		client,
	)

	sender.Run()

	assert.Equal(t, 1, requestCount, "Run() must send exactly one batch request")

	for _, metricName := range GaugeMetrics {
		t.Run(fmt.Sprintf("Received %v", metricName), func(t *testing.T) {
			assert.Equal(t, true, received[metricName])
		})
		t.Run(fmt.Sprintf("Method %v", metricName), func(t *testing.T) {
			assert.Equal(t, http.MethodPost, method[metricName])
		})
		t.Run(fmt.Sprintf("Content-Type %v", metricName), func(t *testing.T) {
			assert.Equal(t, expectedContentType, contentType[metricName])
		})
	}
	for _, metricName := range CounterMetrics {
		t.Run(fmt.Sprintf("Received %v", metricName), func(t *testing.T) {
			assert.Equal(t, true, received[metricName])
		})
		t.Run(fmt.Sprintf("Method %v", metricName), func(t *testing.T) {
			assert.Equal(t, http.MethodPost, method[metricName])
		})
		t.Run(fmt.Sprintf("Content-Type %v", metricName), func(t *testing.T) {
			assert.Equal(t, expectedContentType, contentType[metricName])
		})
	}
}

// TestResetCounterOnSuccess verifies that counter metrics are reset to zero
// only after successful delivery (HTTP 200).
func TestResetCounterOnSuccess(t *testing.T) {
	storage := NewAgentStorage()

	initialValue := rand.Int64()
	for _, metricName := range CounterMetrics {
		storage.AddCounter(metricName, initialValue)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(testHandler)
	defer ts.Close()

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second},
	)
	sender := NewSender(
		ts.URL,
		storage,
		client,
	)

	sender.Run()

	for _, metricName := range CounterMetrics {
		t.Run(fmt.Sprintf("counter %v reset to zero", metricName), func(t *testing.T) {
			value, _ := storage.GetCounter(metricName)
			assert.Equal(t, int64(0), value)
		})
	}
}

// TestKeepCounterOnError verifies that counter metrics are NOT reset
// when delivery fails (non-200 response), preserving them for retry.
func TestKeepCounterOnError(t *testing.T) {
	storage := NewAgentStorage()

	initialValue := rand.Int64()
	for _, metricName := range CounterMetrics {
		storage.AddCounter(metricName, initialValue)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ts := httptest.NewServer(testHandler)
	defer ts.Close()

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second},
	)
	sender := NewSender(
		ts.URL,
		storage,
		client,
	)

	sender.Run()

	for _, metricName := range CounterMetrics {
		t.Run(fmt.Sprintf("counter %v preserved on error", metricName), func(t *testing.T) {
			value, _ := storage.GetCounter(metricName)
			assert.Equal(t, initialValue, value)
		})
	}
}

// TestNoRequestsWhenStorageEmpty verifies that Run() sends nothing
// when the storage holds no metrics.
func TestNoRequestsWhenStorageEmpty(t *testing.T) {
	storage := NewAgentStorage()
	var requestCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second},
	)
	sender := NewSender(
		ts.URL,
		storage,
		client,
	)

	sender.Run()

	assert.Equal(t, int32(0), requestCount.Load(), "no requests expected for empty storage")
}

// TestKeepCounterOnNetworkError verifies that counter metrics are NOT reset
// when the server is unreachable, preserving them for retry.
func TestKeepCounterOnNetworkError(t *testing.T) {
	storage := NewAgentStorage()

	initialValue := rand.Int64()
	for _, metricName := range CounterMetrics {
		storage.AddCounter(metricName, initialValue)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ts.Close() // make the server unreachable

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second},
	)
	sender := NewSender(
		ts.URL,
		storage,
		client,
	)

	sender.Run()

	for _, metricName := range CounterMetrics {
		t.Run(fmt.Sprintf("counter %v preserved on network error", metricName), func(t *testing.T) {
			value, _ := storage.GetCounter(metricName)
			assert.Equal(t, initialValue, value)
		})
	}
}

// retryTransport simulates transient network failures: the first `failures`
// attempts return a transport error, later attempts are delegated to the
// underlying transport. It records a timestamp for every attempt.
type retryTransport struct {
	failures int
	base     http.RoundTripper

	mu       sync.Mutex
	attempts []time.Time
}

// RoundTrip records the attempt and delegates it to the underlying transport.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.attempts = append(t.attempts, time.Now())
	n := len(t.attempts)
	t.mu.Unlock()

	if n <= t.failures {
		return nil, fmt.Errorf("attempt %d: simulated connection reset", n)
	}
	return t.base.RoundTrip(req)
}

// TestRetrySucceedsAfterTransientFailures verifies that the sender retries
// transport errors and delivers the batch on a later attempt; the retried
// request must carry an intact gzip-compressed JSON body.
func TestRetrySucceedsAfterTransientFailures(t *testing.T) {
	storage := NewAgentStorage()
	storage.SetGauge("retry_gauge", 42.5)
	storage.AddCounter("retry_counter", 7)

	var mu sync.Mutex
	received := map[string]bool{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer gz.Close()

		var metrics []model.Metric
		if err := json.NewDecoder(gz).Decode(&metrics); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		for _, metric := range metrics {
			received[metric.ID] = true
		}
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	rt := &retryTransport{failures: 2, base: http.DefaultTransport}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second, Transport: rt},
	)
	sender := NewSender(
		ts.URL,
		storage,
		client,
	)

	sender.Run()

	assert.Equal(t, 3, len(rt.attempts), "two failed attempts plus one successful retry expected")
	assert.True(t, received["retry_gauge"], "gauge must be delivered after retries")
	assert.True(t, received["retry_counter"], "counter must be delivered after retries")

	value, _ := storage.GetCounter("retry_counter")
	assert.Equal(t, int64(0), value, "counter must be reset after successful retry delivery")
}

// TestRetrySucceedsOnLastAttempt verifies that the batch is delivered when
// only the final allowed retry succeeds.
func TestRetrySucceedsOnLastAttempt(t *testing.T) {
	storage := NewAgentStorage()
	storage.AddCounter("last_chance_counter", 3)

	var requestCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	rt := &retryTransport{failures: len(retries), base: http.DefaultTransport}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second, Transport: rt},
	)

	sender := NewSender(ts.URL, storage, client)

	sender.Run()

	assert.Equal(t, int32(1), requestCount.Load(), "exactly one request must reach the server")
	value, _ := storage.GetCounter("last_chance_counter")
	assert.Equal(t, int64(0), value, "counter must be reset when the last retry succeeds")
}

// TestRetryExhaustedRestoresCounters verifies that after exhausting all
// retries the drained counters are merged back into storage and that the
// configured backoff intervals are respected between attempts.
func TestRetryExhaustedRestoresCounters(t *testing.T) {
	storage := NewAgentStorage()
	storage.AddCounter("exhausted_counter", 11)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be reached when every attempt fails")
	}))
	defer ts.Close()

	retries := []time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 15 * time.Millisecond}
	rt := &retryTransport{failures: 100, base: http.DefaultTransport}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second, Transport: rt},
	)

	sender := NewSender(ts.URL, storage, client)

	sender.Run()

	assert.Equal(t, len(retries)+1, len(rt.attempts), "initial attempt plus one retry per timeout expected")

	value, _ := storage.GetCounter("exhausted_counter")
	assert.Equal(t, int64(11), value, "counter must be restored after exhausted retries")

	for i := 1; i < len(rt.attempts); i++ {
		gap := rt.attempts[i].Sub(rt.attempts[i-1])
		assert.GreaterOrEqual(t, gap, retries[i-1], "retry %d must wait at least %v", i, retries[i-1])
	}
}

// TestNoRetryOnServerErrorResponse documents that HTTP-level errors are not
// retried: the batch is dropped until the next report interval and the
// drained counters are preserved.
func TestNoRetryOnServerErrorResponse(t *testing.T) {
	storage := NewAgentStorage()
	storage.AddCounter("server_error_counter", 5)

	var requestCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	retries := []time.Duration{1 * time.Millisecond, 3 * time.Millisecond, 5 * time.Millisecond}
	client := NewClientWithRetries(
		retries,
		&http.Client{Timeout: 5 * time.Second},
	)
	sender := NewSender(
		ts.URL,
		storage,
		client,
	)

	sender.Run()

	assert.Equal(t, int32(1), requestCount.Load(), "server error responses must not be retried")
	value, _ := storage.GetCounter("server_error_counter")
	assert.Equal(t, int64(5), value, "counter must be preserved on server error response")
}
