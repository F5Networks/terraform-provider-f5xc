//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceKeyForCustomRPC verifies that the RPC method is dropped so all
// RPCs of one service share a throttle key.
func TestResourceKeyForCustomRPC(t *testing.T) {
	tests := []struct {
		fqn  string
		want string
	}{
		{fqn: "", want: ""},
		{fqn: "   ", want: ""},
		{fqn: "SetState", want: "setstate"},
		{fqn: "ves.io.schema.site.CustomStateAPI.SetState", want: "ves.io.schema.site.customstateapi"},
		{fqn: "  Ves.IO.Schema.NS.CustomAPI.CascadeDelete  ", want: "ves.io.schema.ns.customapi"},
	}

	for _, tc := range tests {
		t.Run(tc.fqn, func(t *testing.T) {
			assert.Equal(t, tc.want, resourceKeyForCustomRPC(tc.fqn))
		})
	}
}

// TestNormalizeResourceKey verifies case and whitespace folding, and the
// fallback key for empty input.
func TestNormalizeResourceKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: defaultResourceKey},
		{in: "  ", want: defaultResourceKey},
		{in: "HTTP_LoadBalancers", want: "http_loadbalancers"},
		{in: " origin_pools ", want: "origin_pools"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeResourceKey(tc.in))
		})
	}
}

// TestDeriveResourceKeyFromPath verifies that the segment after the namespace
// name is the key, and that other paths fall back to the last segment.
func TestDeriveResourceKeyFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: defaultResourceKey},
		{name: "root", path: "/", want: defaultResourceKey},
		{name: "object path", path: "/api/config/namespaces/default/http_loadbalancers/lb1", want: "http_loadbalancers"},
		{name: "collection path", path: "/api/config/namespaces/default/origin_pools", want: "origin_pools"},
		{name: "namespace only", path: "/api/web/namespaces/default", want: "default"},
		{name: "no namespaces segment", path: "/api/web/custom/Tenant/Settings", want: "settings"},
		{name: "trailing slash", path: "/api/config/namespaces/system/sites/", want: "sites"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, deriveResourceKeyFromPath(tc.path))
		})
	}
}

// TestResolveRequestResourceKey verifies that a key in the request context
// takes precedence over the key derived from the URL path.
func TestResolveRequestResourceKey(t *testing.T) {
	const path = "http://example.test/api/config/namespaces/default/origin_pools/op1"

	tests := []struct {
		name string
		req  func() *http.Request
		want string
	}{
		{name: "nil request", req: func() *http.Request { return nil }, want: defaultResourceKey},
		{
			name: "derived from path",
			req:  func() *http.Request { return httptest.NewRequest(http.MethodGet, path, nil) },
			want: "origin_pools",
		},
		{
			name: "context key wins",
			req: func() *http.Request {
				ctx := contextFromResourceKey(context.Background(), "Virtual_Sites")
				return httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
			},
			want: "virtual_sites",
		},
		{
			name: "empty context key falls back to path",
			req: func() *http.Request {
				ctx := contextFromResourceKey(context.Background(), "")
				return httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
			},
			want: "origin_pools",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveRequestResourceKey(tc.req()))
		})
	}
}

// TestParseRetryAfterDelay_Cases verifies the accepted Retry-After forms and
// that invalid values are rejected.
func TestParseRetryAfterDelay_Cases(t *testing.T) {
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)

	tests := []struct {
		name    string
		header  *string
		wantOK  bool
		wantMin time.Duration
		wantMax time.Duration
	}{
		{name: "nil response", header: nil},
		{name: "missing header", header: new("")},
		{name: "zero seconds", header: new("0"), wantOK: true},
		{name: "negative seconds", header: new("-5")},
		{name: "garbage", header: new("soon")},
		{name: "whitespace around seconds", header: new(" 7 "), wantOK: true, wantMin: 7 * time.Second, wantMax: 7 * time.Second},
		{name: "future http date", header: new(future), wantOK: true, wantMin: 25 * time.Second, wantMax: 30 * time.Second},
		{name: "past http date", header: new(past), wantOK: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			if tc.header != nil {
				resp = &http.Response{Header: http.Header{}}
				if *tc.header != "" {
					resp.Header.Set("Retry-After", *tc.header)
				}
			}

			got, ok := parseRetryAfterDelay(resp)

			assert.Equal(t, tc.wantOK, ok)
			assert.GreaterOrEqual(t, got, tc.wantMin)
			assert.LessOrEqual(t, got, tc.wantMax)
		})
	}
}

// TestRetryDelayForAttempt verifies that a positive Retry-After is used only
// when the policy respects it; otherwise exponential backoff applies.
func TestRetryDelayForAttempt(t *testing.T) {
	policy := throttlingPolicy{
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     time.Second,
		Multiplier:     2,
	}
	withRetryAfter := func(v string) *http.Response {
		return &http.Response{Header: http.Header{"Retry-After": []string{v}}}
	}

	tests := []struct {
		name    string
		respect bool
		attempt int
		resp    *http.Response
		want    time.Duration
	}{
		{name: "retry-after respected", respect: true, attempt: 1, resp: withRetryAfter("3"), want: 3 * time.Second},
		{name: "retry-after ignored", respect: false, attempt: 1, resp: withRetryAfter("3"), want: 100 * time.Millisecond},
		{name: "zero retry-after uses backoff", respect: true, attempt: 2, resp: withRetryAfter("0"), want: 200 * time.Millisecond},
		{name: "no response uses backoff", respect: true, attempt: 3, resp: nil, want: 400 * time.Millisecond},
		{name: "backoff capped at max", respect: true, attempt: 10, resp: nil, want: time.Second},
		{name: "attempt below one treated as one", respect: true, attempt: 0, resp: nil, want: 100 * time.Millisecond},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := policy
			p.RespectRetryAfter = tc.respect
			assert.Equal(t, tc.want, retryDelayForAttempt(tc.attempt, p, tc.resp))
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

// TestEnsureReplayableBody verifies that a request body can be read again for
// every retry attempt, and that a body read error is returned.
func TestEnsureReplayableBody(t *testing.T) {
	t.Run("nil request and nil body", func(t *testing.T) {
		ok, err := ensureReplayableBody(nil)
		require.NoError(t, err)
		assert.True(t, ok)

		req := httptest.NewRequest(http.MethodGet, "http://example.test", nil)
		req.Body = nil
		ok, err = ensureReplayableBody(req)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("existing GetBody is kept", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "http://example.test", bytes.NewBufferString("x"))
		require.NoError(t, err)
		require.NotNil(t, req.GetBody)

		ok, err := ensureReplayableBody(req)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("opaque body becomes replayable", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, "http://example.test", io.NopCloser(strings.NewReader(`{"a":1}`)))
		require.NoError(t, err)
		require.Nil(t, req.GetBody)

		ok, err := ensureReplayableBody(req)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, int64(len(`{"a":1}`)), req.ContentLength)

		for attempt := 1; attempt <= 3; attempt++ {
			clone, cloneErr := cloneRequestForAttempt(req)
			require.NoError(t, cloneErr)
			body, readErr := io.ReadAll(clone.Body)
			require.NoError(t, readErr)
			assert.Equal(t, `{"a":1}`, string(body), "attempt %d", attempt)
		}
	})

	t.Run("read error", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, "http://example.test", io.NopCloser(failingReader{}))
		require.NoError(t, err)

		ok, err := ensureReplayableBody(req)
		require.Error(t, err)
		assert.False(t, ok)
	})
}

// TestCloneRequestForAttempt_NoBody verifies that a request without GetBody is
// cloned with the same method, URL, and context.
func TestCloneRequestForAttempt_NoBody(t *testing.T) {
	ctx := contextFromResourceKey(context.Background(), "sites")
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api/x", nil).WithContext(ctx)
	req.GetBody = nil

	clone, err := cloneRequestForAttempt(req)

	require.NoError(t, err)
	assert.NotSame(t, req, clone)
	assert.Equal(t, req.Method, clone.Method)
	assert.Equal(t, req.URL.String(), clone.URL.String())
	assert.Equal(t, "sites", resourceKeyFromContext(clone.Context()))
}

// TestResourceThrottleRegistry_NilAndPublish verifies that a nil registry never
// blocks, and that publish keeps the latest cooldown instead of shortening it.
func TestResourceThrottleRegistry_NilAndPublish(t *testing.T) {
	var nilRegistry *ResourceThrottleRegistry
	require.NoError(t, nilRegistry.wait(context.Background(), "x"))
	nilRegistry.publish("x", time.Second)

	r := newResourceThrottleRegistry()
	r.publish("Sites", time.Hour)
	long := r.coordinator("sites").nextAllowedAt.Load()
	require.NotZero(t, long, "keys are normalized before lookup")

	r.publish("sites", time.Millisecond)
	assert.Equal(t, long, r.coordinator("sites").nextAllowedAt.Load(), "a shorter delay must not shorten the cooldown")

	r.publish("other", 0)
	assert.Zero(t, r.coordinator("other").nextAllowedAt.Load(), "non-positive delay is ignored")
}

// fastRetryPolicy returns an enabled policy with millisecond backoff and no
// jitter, so retry tests run quickly and deterministically.
func fastRetryPolicy(maxRetries int) throttlingPolicy {
	p := defaultThrottlingPolicy()
	p.Enabled = true
	p.MaxRetries = maxRetries
	p.InitialBackoff = time.Millisecond
	p.MaxBackoff = 2 * time.Millisecond
	p.JitterRatio = 0
	p.RespectRetryAfter = false
	return p
}

func statusResponse(code int) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(http.StatusText(code))), Header: http.Header{}}
}

func newTestRequest(t *testing.T, method string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method,
		"https://example.test/api/config/namespaces/default/origin_pools/op1", body)
	require.NoError(t, err)
	return req
}

// TestDoRequestWithThrottling_NilRequest verifies that a nil request is
// rejected before any send.
func TestDoRequestWithThrottling_NilRequest(t *testing.T) {
	sends := 0
	resp, err := doRequestWithThrottling(context.Background(), nil, "x", fastRetryPolicy(1), newResourceThrottleRegistry(),
		func(*http.Request) (*http.Response, error) { sends++; return nil, nil })

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Zero(t, sends)
}

// TestDoRequestWithThrottling_Disabled verifies that with throttling disabled,
// or without a registry, a request is sent once and a 429 is returned as is.
func TestDoRequestWithThrottling_Disabled(t *testing.T) {
	disabled := fastRetryPolicy(3)
	disabled.Enabled = false

	tests := []struct {
		name     string
		policy   throttlingPolicy
		registry *ResourceThrottleRegistry
	}{
		{name: "policy disabled", policy: disabled, registry: newResourceThrottleRegistry()},
		{name: "nil registry", policy: fastRetryPolicy(3), registry: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sends := 0
			resp, err := doRequestWithThrottling(context.Background(), newTestRequest(t, http.MethodGet, nil), "origin_pools",
				tc.policy, tc.registry, func(*http.Request) (*http.Response, error) {
					sends++
					return statusResponse(http.StatusTooManyRequests), nil
				})

			require.NoError(t, err)
			defer closeResponseBody(resp)
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
			assert.Equal(t, 1, sends)
		})
	}
}

// TestDoRequestWithThrottling_MaxRetriesExhausted verifies that a resource that
// keeps returning 429 is sent MaxRetries+1 times, and the last 429 is returned.
func TestDoRequestWithThrottling_MaxRetriesExhausted(t *testing.T) {
	for _, maxRetries := range []int{0, 1, 3} {
		t.Run(fmt.Sprintf("max_retries=%d", maxRetries), func(t *testing.T) {
			sends := 0
			resp, err := doRequestWithThrottling(context.Background(), newTestRequest(t, http.MethodGet, nil), "origin_pools",
				fastRetryPolicy(maxRetries), newResourceThrottleRegistry(), func(*http.Request) (*http.Response, error) {
					sends++
					return statusResponse(http.StatusTooManyRequests), nil
				})

			require.NoError(t, err)
			defer closeResponseBody(resp)
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
			assert.Equal(t, maxRetries+1, sends)
		})
	}
}

// TestDoRequestWithThrottling_OnlyRetries429 verifies that other error statuses
// and transport errors are returned after a single send.
func TestDoRequestWithThrottling_OnlyRetries429(t *testing.T) {
	transportErr := errors.New("connection reset")

	tests := []struct {
		name       string
		resp       *http.Response
		err        error
		wantStatus int
	}{
		{name: "500 not retried", resp: statusResponse(http.StatusInternalServerError), wantStatus: http.StatusInternalServerError},
		{name: "503 not retried", resp: statusResponse(http.StatusServiceUnavailable), wantStatus: http.StatusServiceUnavailable},
		{name: "404 not retried", resp: statusResponse(http.StatusNotFound), wantStatus: http.StatusNotFound},
		{name: "transport error not retried", err: transportErr},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sends := 0
			resp, err := doRequestWithThrottling(context.Background(), newTestRequest(t, http.MethodGet, nil), "origin_pools",
				fastRetryPolicy(3), newResourceThrottleRegistry(), func(*http.Request) (*http.Response, error) {
					sends++
					return tc.resp, tc.err
				})

			assert.Equal(t, 1, sends)
			if tc.err != nil {
				require.ErrorIs(t, err, transportErr)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			defer closeResponseBody(resp)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

// TestDoRequestWithThrottling_RetriesReplayBody verifies that every retry sends
// the full original request body.
func TestDoRequestWithThrottling_RetriesReplayBody(t *testing.T) {
	const payload = `{"spec":{"port":443}}`
	var bodies []string

	resp, err := doRequestWithThrottling(context.Background(),
		newTestRequest(t, http.MethodPut, io.NopCloser(strings.NewReader(payload))), "origin_pools",
		fastRetryPolicy(2), newResourceThrottleRegistry(), func(r *http.Request) (*http.Response, error) {
			b, readErr := io.ReadAll(r.Body)
			require.NoError(t, readErr)
			bodies = append(bodies, string(b))
			if len(bodies) < 3 {
				return statusResponse(http.StatusTooManyRequests), nil
			}
			return statusResponse(http.StatusOK), nil
		})

	require.NoError(t, err)
	defer closeResponseBody(resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, []string{payload, payload, payload}, bodies)
}

// TestDoRequestWithThrottling_UnreadableBody verifies that a body read failure
// is returned before any send.
func TestDoRequestWithThrottling_UnreadableBody(t *testing.T) {
	sends := 0
	resp, err := doRequestWithThrottling(context.Background(),
		newTestRequest(t, http.MethodPut, io.NopCloser(failingReader{})), "origin_pools",
		fastRetryPolicy(2), newResourceThrottleRegistry(), func(*http.Request) (*http.Response, error) {
			sends++
			return statusResponse(http.StatusOK), nil
		})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Zero(t, sends)
}

// TestDoRequestWithThrottling_PublishesCooldown verifies that a 429 publishes a
// cooldown, so other requests to the same resource wait.
func TestDoRequestWithThrottling_PublishesCooldown(t *testing.T) {
	policy := fastRetryPolicy(1)
	policy.InitialBackoff = 30 * time.Millisecond
	policy.MaxBackoff = 30 * time.Millisecond
	registry := newResourceThrottleRegistry()

	sends := 0
	resp, err := doRequestWithThrottling(context.Background(), newTestRequest(t, http.MethodGet, nil), "origin_pools",
		policy, registry, func(*http.Request) (*http.Response, error) {
			sends++
			if sends == 1 {
				return statusResponse(http.StatusTooManyRequests), nil
			}
			return statusResponse(http.StatusOK), nil
		})
	require.NoError(t, err)
	closeResponseBody(resp)

	assert.NotZero(t, registry.coordinator("origin_pools").nextAllowedAt.Load(), "429 must publish a cooldown")
	assert.Zero(t, registry.coordinator("http_loadbalancers").nextAllowedAt.Load(), "other resources are not throttled")
}

// TestDoRequestWithThrottling_CancelledDuringBackoff verifies that cancelling
// the context while waiting between retries stops further sends.
func TestDoRequestWithThrottling_CancelledDuringBackoff(t *testing.T) {
	policy := fastRetryPolicy(3)
	policy.InitialBackoff = 5 * time.Second
	policy.MaxBackoff = 5 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sends := 0
	start := time.Now()
	resp, err := doRequestWithThrottling(ctx, newTestRequest(t, http.MethodGet, nil), "origin_pools",
		policy, newResourceThrottleRegistry(), func(*http.Request) (*http.Response, error) {
			sends++
			time.AfterFunc(10*time.Millisecond, cancel)
			return statusResponse(http.StatusTooManyRequests), nil
		})

	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, resp)
	assert.Equal(t, 1, sends)
	assert.Less(t, time.Since(start), 2*time.Second, "must not wait for the full backoff")
}

// TestDoRequestWithThrottling_DeadlineDuringCooldown verifies that an expired
// deadline while waiting for another request's cooldown returns before any send.
func TestDoRequestWithThrottling_DeadlineDuringCooldown(t *testing.T) {
	registry := newResourceThrottleRegistry()
	registry.publish("origin_pools", 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	sends := 0
	_, err := doRequestWithThrottling(ctx, newTestRequest(t, http.MethodGet, nil), "origin_pools",
		fastRetryPolicy(1), registry, func(*http.Request) (*http.Response, error) {
			sends++
			return statusResponse(http.StatusOK), nil
		})

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Zero(t, sends)
}

// TestResourceThrottleRegistry_ExpiredCooldown verifies that wait returns at
// once when the published cooldown has already passed.
func TestResourceThrottleRegistry_ExpiredCooldown(t *testing.T) {
	r := newResourceThrottleRegistry()
	r.coordinator("sites").nextAllowedAt.Store(time.Now().Add(-time.Second).UnixNano())

	start := time.Now()
	require.NoError(t, r.wait(context.Background(), "sites"))
	assert.Less(t, time.Since(start), 20*time.Millisecond)
}

// TestComputeBackoffDelay_Jitter verifies that jitter keeps every delay within
// ±JitterRatio of the base delay and never above MaxBackoff.
func TestComputeBackoffDelay_Jitter(t *testing.T) {
	policy := throttlingPolicy{
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     time.Second,
		Multiplier:     2,
		JitterRatio:    0.2,
	}

	tests := []struct {
		attempt int
		base    time.Duration
	}{
		{attempt: 1, base: 100 * time.Millisecond},
		{attempt: 3, base: 400 * time.Millisecond},
		{attempt: 10, base: time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.base.String(), func(t *testing.T) {
			lo := time.Duration(float64(tc.base) * 0.8)
			hi := min(time.Duration(float64(tc.base)*1.2), policy.MaxBackoff)
			for i := 0; i < 1000; i++ {
				d := computeBackoffDelay(tc.attempt, policy)
				require.GreaterOrEqual(t, d, lo)
				require.LessOrEqual(t, d, hi)
			}
		})
	}
}

// TestResolveThrottlingPolicy_Validation verifies that provider overrides are
// applied and that invalid throttling settings are rejected.
func TestResolveThrottlingPolicy_Validation(t *testing.T) {
	t.Run("overrides applied", func(t *testing.T) {
		p, err := resolveThrottlingPolicy(&ThrottlingConfig{
			MaxRetries: 7, InitialBackoff: "50ms", MaxBackoff: "3s",
			Multiplier: 1.5, JitterRatio: 0.5, RespectRetryAfter: true,
		})
		require.NoError(t, err)
		assert.Equal(t, throttlingPolicy{
			Enabled: true, MaxRetries: 7, InitialBackoff: 50 * time.Millisecond, MaxBackoff: 3 * time.Second,
			Multiplier: 1.5, JitterRatio: 0.5, RespectRetryAfter: true,
		}, p)
	})

	tests := []struct {
		name    string
		cfg     ThrottlingConfig
		wantErr string
	}{
		{name: "negative retries", cfg: ThrottlingConfig{MaxRetries: -1}, wantErr: "max_retries"},
		{name: "bad initial backoff", cfg: ThrottlingConfig{InitialBackoff: "soon"}, wantErr: "initial_backoff"},
		{name: "bad max backoff", cfg: ThrottlingConfig{MaxBackoff: "later"}, wantErr: "max_backoff"},
		{name: "zero initial backoff", cfg: ThrottlingConfig{InitialBackoff: "0s"}, wantErr: "initial_backoff must be > 0"},
		{name: "negative max backoff", cfg: ThrottlingConfig{MaxBackoff: "-1s"}, wantErr: "max_backoff must be > 0"},
		{name: "max below initial", cfg: ThrottlingConfig{InitialBackoff: "2s", MaxBackoff: "1s"}, wantErr: "max_backoff must be >= throttling.initial_backoff"},
		{name: "multiplier below one", cfg: ThrottlingConfig{Multiplier: 0.5}, wantErr: "multiplier"},
		{name: "jitter above one", cfg: ThrottlingConfig{JitterRatio: 1.5}, wantErr: "jitter_ratio"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveThrottlingPolicy(&tc.cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestCloseResponseBody verifies that nil responses and nil bodies are ignored,
// and that a body is drained and closed.
func TestCloseResponseBody(t *testing.T) {
	assert.NotPanics(t, func() { closeResponseBody(nil) })
	assert.NotPanics(t, func() { closeResponseBody(&http.Response{}) })

	body := &trackingBody{Reader: strings.NewReader("remaining")}
	closeResponseBody(&http.Response{Body: body})
	assert.True(t, body.closed)
	assert.Zero(t, body.Len(), "body must be drained before close")
}

type trackingBody struct {
	*strings.Reader
	closed bool
}

func (b *trackingBody) Close() error { b.closed = true; return nil }

type failingCloseBody struct{ io.Reader }

func (failingCloseBody) Close() error { return errors.New("close failed") }

// TestEnsureReplayableBody_CloseError verifies that a failure to close the
// original body is returned, because the request cannot be replayed safely.
func TestEnsureReplayableBody_CloseError(t *testing.T) {
	req := newTestRequest(t, http.MethodPut, failingCloseBody{strings.NewReader("x")})
	req.GetBody = nil

	ok, err := ensureReplayableBody(req)

	require.EqualError(t, err, "close failed")
	assert.False(t, ok)
}

// TestCloneRequestForAttempt_GetBodyError verifies that a body that cannot be
// rebuilt for a retry returns an error instead of sending an empty body.
func TestCloneRequestForAttempt_GetBodyError(t *testing.T) {
	req := newTestRequest(t, http.MethodPut, strings.NewReader("x"))
	req.GetBody = func() (io.ReadCloser, error) { return nil, errors.New("body gone") }

	clone, err := cloneRequestForAttempt(req)

	require.EqualError(t, err, "body gone")
	assert.Nil(t, clone)
}
