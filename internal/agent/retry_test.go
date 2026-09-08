package agent

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ HTTPClient = (*fakeHTTPClient)(nil)

// fakeHTTPClient simulates a flaky HTTPClient: every queued error is
// returned for the corresponding Do call, and once the queue is exhausted
// it answers with a synthetic 200 OK response. It counts the total number
// of Do calls.
type fakeHTTPClient struct {
	errors []error

	mu    sync.Mutex
	calls int
}

// Do returns the next queued error, or a synthetic success response once
// the error queue is exhausted.
func (c *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	n := c.calls
	c.calls++
	c.mu.Unlock()

	if n < len(c.errors) {
		return nil, c.errors[n]
	}
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

// newRetryRequest builds a POST request with a rewindable body, which the
// retry wrapper restores between attempts.
func newRetryRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(
		http.MethodPost,
		"http://retry.test/updates",
		bytes.NewReader([]byte("{}")),
	)
	require.NoError(t, err)
	return req
}

// TestClientWithRetries_SucceedsAfterTwoFailures verifies that the wrapper
// retries transport errors and returns the first successful response.
func TestClientWithRetries_SucceedsAfterTwoFailures(t *testing.T) {
	fake := &fakeHTTPClient{errors: []error{
		errors.New("attempt 1: simulated connection reset"),
		errors.New("attempt 2: simulated connection reset"),
	}}
	client := NewClientWithRetries(
		[]time.Duration{time.Millisecond, time.Millisecond},
		fake,
	)

	resp, err := client.Do(newRetryRequest(t))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, fake.calls, "two failed attempts plus one successful retry expected")
}

// TestClientWithRetries_RetriesExhausted verifies that the wrapper gives up
// after one retry per configured timeout and returns the last transport error.
func TestClientWithRetries_RetriesExhausted(t *testing.T) {
	errs := make([]error, 4)
	for i := range errs {
		errs[i] = fmt.Errorf("attempt %d: simulated connection reset", i+1)
	}
	fake := &fakeHTTPClient{errors: errs}
	client := NewClientWithRetries(
		[]time.Duration{time.Millisecond, time.Millisecond, time.Millisecond},
		fake,
	)

	resp, err := client.Do(newRetryRequest(t))

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, errs[len(errs)-1], err, "the last transport error must be returned")
	assert.Equal(t, 4, fake.calls, "initial attempt plus one retry per timeout expected")
}

// TestClientWithRetries_SucceedsOnFirstAttempt verifies that a successful
// first call is returned as is, without triggering any retries.
func TestClientWithRetries_SucceedsOnFirstAttempt(t *testing.T) {
	fake := &fakeHTTPClient{}
	client := NewClientWithRetries(
		[]time.Duration{time.Millisecond, time.Millisecond, time.Millisecond},
		fake,
	)

	resp, err := client.Do(newRetryRequest(t))

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, fake.calls, "a successful first attempt must not be retried")
}

// TestClientWithRetries_NoRetryOnNonRewindableBody verifies that a transport
// error on a request without a GetBody function is returned immediately
// instead of panicking on the nil function call during a retry.
func TestClientWithRetries_NoRetryOnNonRewindableBody(t *testing.T) {
	fake := &fakeHTTPClient{errors: []error{
		errors.New("attempt 1: simulated connection reset"),
	}}
	client := NewClientWithRetries(
		[]time.Duration{time.Millisecond, time.Millisecond, time.Millisecond},
		fake,
	)

	// The anonymous struct hides *strings.Reader from http.NewRequest, so
	// the request is built without a GetBody function.
	req, err := http.NewRequest(
		http.MethodPost,
		"http://retry.test/updates",
		struct{ io.Reader }{strings.NewReader("{}")},
	)
	require.NoError(t, err)

	resp, err := client.Do(req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, 1, fake.calls, "a request without a rewindable body must not be retried")
}
