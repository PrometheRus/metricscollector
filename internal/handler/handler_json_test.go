package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handlermock "github.com/nikitaw13/metricscollector/internal/handler/mock"
	"github.com/nikitaw13/metricscollector/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// expectedJSONContentType is the expected Content-Type header for all JSON responses.
const expectedJSONContentType = "application/json; charset=utf-8"

// testJSONRequest sends an HTTP request with a JSON body to the test server
// and returns the full response along with the response body as a string.
func testJSONRequest(t *testing.T, ts *httptest.Server, method, path, body string, contentType string) (*http.Response, string) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, ts.URL+path, bodyReader)
	require.NoError(t, err)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(bodyBytes)
}

// jsonTestCase holds a single table-driven test case for JSON endpoint tests.
type jsonTestCase struct {
	name        string
	method      string
	path        string
	body        string
	contentType string
	want        jsonTestWant
}

// jsonTestWant describes the expected HTTP response for a JSON test case.
type jsonTestWant struct {
	code        int
	contentType string
}

// jsonUpdateTests covers successful POST /update calls (valid counter and gauge payloads).
var jsonUpdateTests = []jsonTestCase{
	{
		"Counter positive int",
		http.MethodPost,
		"/update",
		"{\"type\":\"counter\", \"id\":\"test\", \"delta\":1000}",
		"application/json",
		jsonTestWant{
			http.StatusOK,
			expectedJSONContentType,
		},
	},
	{
		"Gauge positive int",
		http.MethodPost,
		"/update",
		`{"type":"gauge", "id":"test", "value":1000}`,
		"application/json",
		jsonTestWant{
			http.StatusOK,
			expectedJSONContentType,
		},
	},
	{
		"Gauge negative int",
		http.MethodPost,
		"/update",
		`{"type":"gauge", "id":"test", "value":-1000}`,
		"application/json",
		jsonTestWant{
			http.StatusOK,
			expectedJSONContentType,
		},
	},
	{
		"Gauge positive float",
		http.MethodPost,
		"/update",
		`{"type":"gauge", "id":"test", "value":1000.00}`,
		"application/json",
		jsonTestWant{
			http.StatusOK,
			expectedJSONContentType,
		},
	},
	{
		"Gauge negative float",
		http.MethodPost,
		"/update",
		`{"type":"gauge", "id":"test", "value":-1000.00}`,
		"application/json",
		jsonTestWant{
			http.StatusOK,
			expectedJSONContentType,
		},
	},
}

// jsonValidationTests covers POST /update and POST /value with invalid payloads
// (missing/incorrect type, name, or value) — all expected to return 400 or 404.
var jsonValidationTests = []jsonTestCase{
	// Missing or invalid metric TYPE
	{
		"Missing metric type",
		http.MethodPost,
		"/update",
		"{}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Invalid metric type",
		http.MethodPost,
		"/update",
		"{\"type\":\"random\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},

	// Missing or invalid metric NAME
	{
		"Missing gauge metric name",
		http.MethodPost,
		"/update",
		"{\"type\":\"gauge\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Missing counter metric name",
		http.MethodPost,
		"/update",
		"{\"type\":\"counter\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Invalid gauge metric name",
		http.MethodPost,
		"/value",
		"{\"type\":\"gauge\",\"id\":\"unknown\"}",
		"application/json",
		jsonTestWant{
			http.StatusNotFound,
			expectedJSONContentType,
		},
	},
	{
		"Invalid counter metric name",
		http.MethodPost,
		"/value",
		"{\"type\":\"counter\",\"id\":\"unknown\"}",
		"application/json",
		jsonTestWant{
			http.StatusNotFound,
			expectedJSONContentType,
		},
	},
	{
		"Value invalid JSON syntax",
		http.MethodPost,
		"/value",
		"{\"type\":\"gauge\", \"id\":",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Value missing metric type",
		http.MethodPost,
		"/value",
		"{}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Value invalid metric type",
		http.MethodPost,
		"/value",
		"{\"type\":\"random\",\"id\":\"test\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Value missing metric name",
		http.MethodPost,
		"/value",
		"{\"type\":\"gauge\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},

	// Missing or invalid metric VALUE
	{
		"Missing gauge metric value",
		http.MethodPost,
		"/update",
		"{\"type\":\"gauge\", \"id\":\"name\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Missing counter metric value",
		http.MethodPost,
		"/update",
		"{\"type\":\"counter\", \"id\":\"name\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Gauge metric value is latin",
		http.MethodPost,
		"/update",
		"{\"type\":\"gauge\", \"id\":\"name\", \"value\":\"LatinText\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Counter metric value is latin",
		http.MethodPost,
		"/update",
		"{\"type\":\"counter\", \"id\":\"name\", \"delta\":\"LatinText\"}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Counter metric value is positive float",
		http.MethodPost,
		"/update",
		"{\"type\":\"counter\", \"id\":\"name\", \"delta\":1001.00}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Counter metric value is negative float",
		http.MethodPost,
		"/update",
		"{\"type\":\"counter\", \"id\":\"name\", \"delta\":-1001.00}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
	{
		"Counter metric value is negative int",
		http.MethodPost,
		"/update",
		"{\"type\":\"counter\", \"id\":\"test\", \"delta\":-1000}",
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},

	// Invalid JSON syntax
	{
		"Invalid JSON syntax",
		http.MethodPost,
		"/update",
		`{invalid json`,
		"application/json",
		jsonTestWant{
			http.StatusBadRequest,
			expectedJSONContentType,
		},
	},
}

// runJSONTests executes a slice of table-driven JSON test cases against a test server.
func runJSONTests(t *testing.T, cases []jsonTestCase) {
	ts := GetTestServer()
	defer ts.Close()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, _ := testJSONRequest(t, ts, tc.method, tc.path, tc.body, tc.contentType) //nolint:bodyclose // testJSONRequest closes the body
			assert.Equal(t, tc.want.code, resp.StatusCode)
			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

// TestJSONValidate verifies that /update and /value reject invalid JSON payloads.
func TestJSONValidate(t *testing.T) {
	runJSONTests(t, jsonValidationTests)
}

// TestJSONUpdate verifies that /update accepts valid metric payloads and returns 200.
func TestJSONUpdate(t *testing.T) {
	runJSONTests(t, jsonUpdateTests)
}

// jsonUpdatesTests covers POST /updates: batch metric updates. Successful
// batches return 204 without a Content-Type header; validation failures
// return a JSON error body.
var jsonUpdatesTests = []jsonTestCase{
	{
		"Valid batch with gauge and counter",
		http.MethodPost,
		"/updates/",
		`[{"type":"gauge","id":"batch_gauge","value":42.5},{"type":"counter","id":"batch_counter","delta":7}]`,
		"application/json",
		jsonTestWant{http.StatusNoContent, ""},
	},
	{
		"Empty batch returns no content",
		http.MethodPost,
		"/updates/",
		`[]`,
		"application/json",
		jsonTestWant{http.StatusNoContent, ""},
	},
	{
		"Batch with invalid metric type",
		http.MethodPost,
		"/updates/",
		`[{"type":"gauge","id":"batch_gauge","value":1.0},{"type":"random","id":"batch_unknown"}]`,
		"application/json",
		jsonTestWant{http.StatusBadRequest, expectedJSONContentType},
	},
	{
		"Batch with missing gauge value",
		http.MethodPost,
		"/updates/",
		`[{"type":"gauge","id":"batch_gauge"}]`,
		"application/json",
		jsonTestWant{http.StatusBadRequest, expectedJSONContentType},
	},
	{
		"Batch with missing metric name",
		http.MethodPost,
		"/updates/",
		`[{"type":"counter","delta":5}]`,
		"application/json",
		jsonTestWant{http.StatusBadRequest, expectedJSONContentType},
	},
	{
		"Non-array JSON body",
		http.MethodPost,
		"/updates/",
		`{"type":"gauge","id":"batch_gauge","value":1.0}`,
		"application/json",
		jsonTestWant{http.StatusBadRequest, expectedJSONContentType},
	},
	{
		"Invalid JSON syntax",
		http.MethodPost,
		"/updates/",
		`[{invalid json`,
		"application/json",
		jsonTestWant{http.StatusBadRequest, expectedJSONContentType},
	},
	{
		"Empty request body",
		http.MethodPost,
		"/updates/",
		"",
		"application/json",
		jsonTestWant{http.StatusBadRequest, expectedJSONContentType},
	},
}

// TestJSONUpdates verifies that POST /updates accepts valid metric batches
// and rejects invalid payloads with 400.
func TestJSONUpdates(t *testing.T) {
	ts := GetTestServer()
	defer ts.Close()

	for _, tc := range jsonUpdatesTests {
		t.Run(tc.name, func(t *testing.T) {
			resp, _ := testJSONRequest(t, ts, tc.method, tc.path, tc.body, tc.contentType) //nolint:bodyclose // testJSONRequest closes the body
			assert.Equal(t, tc.want.code, resp.StatusCode)
			// 204 responses carry no body, so no Content-Type is set.
			if tc.want.contentType != "" {
				assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
			}
		})
	}
}

// TestJSONUpdates_PersistsBatch verifies that a valid batch is fully stored
// and readable via POST /value.
func TestJSONUpdates_PersistsBatch(t *testing.T) {
	ts := GetTestServer()
	defer ts.Close()

	batch := `[{"type":"gauge","id":"batch_gauge","value":42.5},{"type":"counter","id":"batch_counter","delta":7}]`
	resp, _ := testJSONRequest(t, ts, http.MethodPost, "/updates/", batch, "application/json") //nolint:bodyclose // testJSONRequest closes the body
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	tests := []struct {
		name string
		body string
		want model.Metric
	}{
		{
			"Read batch gauge",
			`{"type":"gauge","id":"batch_gauge"}`,
			model.Metric{ID: "batch_gauge", Type: "gauge", Value: ptrFloat64(42.5)},
		},
		{
			"Read batch counter",
			`{"type":"counter","id":"batch_counter"}`,
			model.Metric{ID: "batch_counter", Type: "counter", Delta: ptrInt64(7)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := testJSONRequest(t, ts, http.MethodPost, "/value", tc.body, "application/json") //nolint:bodyclose // testJSONRequest closes the body
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var got model.Metric
			require.NoError(t, json.Unmarshal([]byte(body), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestJSONUpdates_RejectsBatchAtomically verifies that a batch containing an
// invalid metric is rejected entirely: no metric from the batch is stored.
func TestJSONUpdates_RejectsBatchAtomically(t *testing.T) {
	ts := GetTestServer()
	defer ts.Close()

	batch := `[{"type":"gauge","id":"atomic_gauge","value":1.5},{"type":"random","id":"atomic_unknown"}]`
	resp, _ := testJSONRequest(t, ts, http.MethodPost, "/updates/", batch, "application/json") //nolint:bodyclose // testJSONRequest closes the body
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// The valid metric from the rejected batch must not be stored.
	resp, _ = testJSONRequest(t, ts, http.MethodPost, "/value", `{"type":"gauge","id":"atomic_gauge"}`, "application/json") //nolint:bodyclose // testJSONRequest closes the body
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestJSONRead verifies that /value returns the correct stored metric values in JSON.
func TestJSONRead(t *testing.T) {
	ts := GetTestServer()
	defer ts.Close()

	tests := []struct {
		name string
		body string
		want model.Metric
	}{
		{
			"Read existing gauge",
			"{\"type\":\"gauge\", \"id\":\"___test___\"}",
			model.Metric{ID: "___test___", Type: "gauge", Value: ptrFloat64(defaultGaugeValue)},
		},
		{
			"Read existing counter",
			"{\"type\":\"counter\", \"id\":\"___test___\"}",
			model.Metric{ID: "___test___", Type: "counter", Delta: ptrInt64(defaultCounterValue)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := testJSONRequest(t, ts, http.MethodPost, "/value", tc.body, "application/json") //nolint:bodyclose // testJSONRequest closes the body
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, expectedJSONContentType, resp.Header.Get("Content-Type"))

			var got model.Metric
			require.NoError(t, json.Unmarshal([]byte(body), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

// ptrFloat64 returns a pointer to value, used to build *float64 for test expectations.
func ptrFloat64(value float64) *float64 { return &value }

// ptrInt64 returns a pointer to value, used to build *int64 for test expectations.
func ptrInt64(value int64) *int64 { return &value }

// TestJSONStorageErrors verifies that storage failures are mapped to HTTP 500
// across the JSON endpoints, while a wrapped model.ErrMetricNotFound still
// maps to 404 on reads.
func TestJSONStorageErrors(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("storage is unavailable")
	notFoundGauge := fmt.Errorf("gauge err_gauge %w", model.ErrMetricNotFound)
	notFoundCounter := fmt.Errorf("counter err_counter %w", model.ErrMetricNotFound)

	tests := []struct {
		name  string
		setup func(ctrl *gomock.Controller) Repository
		path  string
		body  string
		want  int
	}{
		{
			name: "update gauge storage error returns 500",
			setup: func(ctrl *gomock.Controller) Repository {
				repo := handlermock.NewMockRepository(ctrl)
				repo.EXPECT().SetGauge("err_gauge", 42.5).Return(storageErr)
				return repo
			},
			path: "/update",
			body: `{"type":"gauge","id":"err_gauge","value":42.5}`,
			want: http.StatusInternalServerError,
		},
		{
			name: "update counter storage error returns 500",
			setup: func(ctrl *gomock.Controller) Repository {
				repo := handlermock.NewMockRepository(ctrl)
				repo.EXPECT().AddCounter("err_counter", int64(5)).Return(int64(0), storageErr)
				return repo
			},
			path: "/update",
			body: `{"type":"counter","id":"err_counter","delta":5}`,
			want: http.StatusInternalServerError,
		},
		{
			name: "updates batch storage error returns 500",
			setup: func(ctrl *gomock.Controller) Repository {
				repo := handlermock.NewMockRepository(ctrl)
				repo.EXPECT().UpdateMetrics(gomock.Any(), gomock.Any()).Return(storageErr)
				return repo
			},
			path: "/updates/",
			body: `[{"type":"gauge","id":"err_gauge","value":42.5}]`,
			want: http.StatusInternalServerError,
		},
		{
			name: "value gauge storage error returns 500",
			setup: func(ctrl *gomock.Controller) Repository {
				repo := handlermock.NewMockRepository(ctrl)
				repo.EXPECT().GetGauge("err_gauge").Return(float64(0), storageErr)
				return repo
			},
			path: "/value",
			body: `{"type":"gauge","id":"err_gauge"}`,
			want: http.StatusInternalServerError,
		},
		{
			name: "value gauge metric not found returns 404",
			setup: func(ctrl *gomock.Controller) Repository {
				repo := handlermock.NewMockRepository(ctrl)
				repo.EXPECT().GetGauge("err_gauge").Return(float64(0), notFoundGauge)
				return repo
			},
			path: "/value",
			body: `{"type":"gauge","id":"err_gauge"}`,
			want: http.StatusNotFound,
		},
		{
			name: "value counter storage error returns 500",
			setup: func(ctrl *gomock.Controller) Repository {
				repo := handlermock.NewMockRepository(ctrl)
				repo.EXPECT().GetCounter("err_counter").Return(int64(0), storageErr)
				return repo
			},
			path: "/value",
			body: `{"type":"counter","id":"err_counter"}`,
			want: http.StatusInternalServerError,
		},
		{
			name: "value counter metric not found returns 404",
			setup: func(ctrl *gomock.Controller) Repository {
				repo := handlermock.NewMockRepository(ctrl)
				repo.EXPECT().GetCounter("err_counter").Return(int64(0), notFoundCounter)
				return repo
			},
			path: "/value",
			body: `{"type":"counter","id":"err_counter"}`,
			want: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ts := GetTestServerWithRepository(tc.setup(ctrl))
			t.Cleanup(ts.Close)

			resp, _ := testJSONRequest(t, ts, http.MethodPost, tc.path, tc.body, "application/json") //nolint:bodyclose // testJSONRequest closes the body
			assert.Equal(t, tc.want, resp.StatusCode)
			assert.Equal(t, expectedJSONContentType, resp.Header.Get("Content-Type"))
		})
	}
}
