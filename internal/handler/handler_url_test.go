package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// doRequest sends an HTTP request to the test server and returns the response and body.
func doRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

// testCase holds a single table-driven test case for URL endpoint tests.
type testCase struct {
	name   string
	method string
	path   string
	want   expectedResponse
}

// expectedResponse describes the anticipated HTTP response for a test case.
type expectedResponse struct {
	code        int
	response    string
	contentType string
}

// updateTests covers successful metric updates.
var updateTests = []testCase{
	{
		"Counter positive int", http.MethodPost, "/update/counter/test/1000", expectedResponse{
			http.StatusOK,
			"Metric 'test' updated\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Counter negative int", http.MethodPost, "/update/counter/test/-1001", expectedResponse{
			http.StatusOK,
			"Metric 'test' updated\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Gauge positive int", http.MethodPost, "/update/gauge/test/1000", expectedResponse{
			http.StatusOK,
			"Metric 'test' updated\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Gauge negative int", http.MethodPost, "/update/gauge/test/-1000", expectedResponse{
			http.StatusOK,
			"Metric 'test' updated\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Gauge positive float", http.MethodPost, "/update/gauge/test/1000.00", expectedResponse{
			http.StatusOK,
			"Metric 'test' updated\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Gauge negative float", http.MethodPost, "/update/gauge/test/-1000.00", expectedResponse{
			http.StatusOK,
			"Metric 'test' updated\n",
			"text/plain; charset=utf-8",
		},
	},
}

// readTests covers successful reads of metric values and the metric list.
var readTests = []testCase{
	{
		"Read all metrics", http.MethodGet, "/", expectedResponse{
			http.StatusOK,
			expectedHTMLResponse,
			"text/html; charset=UTF-8",
		},
	},
	{
		"Read existing gauge", http.MethodGet, "/value/gauge/___test___", expectedResponse{
			code:        http.StatusOK,
			response:    "123\n",
			contentType: "text/plain; charset=utf-8",
		},
	},
	{
		"Read existing counter", http.MethodGet, "/value/counter/___test___", expectedResponse{
			code:        http.StatusOK,
			response:    "234\n",
			contentType: "text/plain; charset=utf-8",
		},
	},
}

// validationTests covers requests with an invalid type, name, or value.
var validationTests = []testCase{
	// Missing or invalid metric type
	{
		"Missing metric type", http.MethodPost, "/update", expectedResponse{
			http.StatusBadRequest,
			"Type is required\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Invalid metric type", http.MethodPost, "/update/random/", expectedResponse{
			http.StatusBadRequest,
			"Invalid metric type\n",
			"text/plain; charset=utf-8",
		},
	},

	// Missing or invalid metric name
	{
		"Missing gauge metric name", http.MethodPost, "/update/gauge", expectedResponse{
			http.StatusNotFound,
			"Metric is required\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Missing counter metric name", http.MethodPost, "/update/counter", expectedResponse{
			http.StatusNotFound,
			"Metric is required\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Invalid gauge metric name", http.MethodGet, "/value/gauge/unknown", expectedResponse{
			http.StatusNotFound,
			"gauge unknown not found\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Invalid counter metric name", http.MethodGet, "/value/counter/unknown", expectedResponse{
			http.StatusNotFound,
			"counter unknown not found\n",
			"text/plain; charset=utf-8",
		},
	},

	// Missing or invalid metric value
	{
		"Missing gauge metric value", http.MethodPost, "/update/gauge/name", expectedResponse{
			http.StatusBadRequest,
			"Metric value is required\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Missing counter metric value", http.MethodPost, "/update/counter/name", expectedResponse{
			http.StatusBadRequest,
			"Metric value is required\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Gauge metric value is latin", http.MethodPost, "/update/gauge/name/value", expectedResponse{
			http.StatusBadRequest,
			"Invalid metric value\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Counter metric value is latin", http.MethodPost, "/update/counter/name/value", expectedResponse{
			http.StatusBadRequest,
			"Invalid metric value\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Counter metric value is positive float", http.MethodPost, "/update/counter/name/1000.00", expectedResponse{
			http.StatusBadRequest,
			"Invalid metric value\n",
			"text/plain; charset=utf-8",
		},
	},
	{
		"Counter metric value is negative float", http.MethodPost, "/update/counter/name/-1001,00", expectedResponse{
			http.StatusBadRequest,
			"Invalid metric value\n",
			"text/plain; charset=utf-8",
		},
	},
}

// runTableTests executes table-driven cases against a test server and asserts status, body, and Content-Type.
func runTableTests(t *testing.T, cases []testCase) {
	ts := GetTestServer()
	defer ts.Close()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := doRequest(t, ts, tc.method, tc.path) //nolint:bodyclose // doRequest closes the body
			assert.Equal(t, tc.want.code, resp.StatusCode)
			assert.Equal(t, tc.want.response, body)
			assert.Equal(t, tc.want.contentType, resp.Header.Get("Content-Type"))
		})
	}
}

// TestUpdate verifies that valid URL update requests return 200 and store the metric.
func TestUpdate(t *testing.T) {
	runTableTests(t, updateTests)
}

// TestRead verifies the metric list page and single metric value reads.
func TestRead(t *testing.T) {
	runTableTests(t, readTests)
}

// TestValidate verifies that requests with an invalid type, name, or value are rejected with 4xx.
func TestValidate(t *testing.T) {
	runTableTests(t, validationTests)
}
