package handler

import (
	"net/http"
	"net/http/httptest"

	"github.com/nikitaw13/metricscollector/internal/repository"
)

// testResponse carries the parts of an HTTP response that handler tests
// assert on: status code, headers, and the body read as a string. It keeps
// *http.Response inside the request helpers so the response body never leaks.
type testResponse struct {
	StatusCode int
	Header     http.Header
	body       string
}

// defaultGaugeValue is the pre-seeded gauge value used across handler tests.
const defaultGaugeValue = 123.00

// defaultCounterValue is the pre-seeded counter value used across handler tests.
const defaultCounterValue = 234

// GetTestServer returns an httptest.Server backed by a fresh MemStorage
// pre-populated with one gauge and one counter metric named "___test___".
// The storage also serves as the database pinger, so /ping always succeeds;
// use GetTestServerWithDatabase for DB-dependent tests.
func GetTestServer() (server *httptest.Server) {
	storage := repository.NewMemStorage()
	storage.SetGauge("___test___", defaultGaugeValue)
	storage.AddCounter("___test___", defaultCounterValue)
	metricsHandler := NewMetricsHandler(storage, storage)
	router := metricsHandler.NewRouter()
	server = httptest.NewServer(router)
	return
}

// GetTestServerWithDatabase returns an httptest.Server backed by a fresh MemStorage
// and the provided Database for testing routes that require DB connectivity.
func GetTestServerWithDatabase(db DBPinger) (server *httptest.Server) {
	storage := repository.NewMemStorage()
	metricsHandler := NewMetricsHandler(storage, db)
	router := metricsHandler.NewRouter()
	server = httptest.NewServer(router)
	return
}

// expectedHTMLResponse is the expected HTML page returned by GET / for the seeded storage.
var expectedHTMLResponse = `<html><body>
<h1>List of names and results of all currently known metrics</h1>
<b>___test___</b>:   <code>234</code><br><b>___test___</b>:   <code>123</code><br><hr></body></html>
`

// GetTestServerWithRepository returns an httptest.Server backed by the provided Repository for tests needing a custom storage.
// The database pinger is a fresh in-memory storage, so /ping succeeds without external dependencies.
func GetTestServerWithRepository(repo Repository) *httptest.Server {
	return httptest.NewServer(NewMetricsHandler(repo, repository.NewMemStorage()).NewRouter())
}
