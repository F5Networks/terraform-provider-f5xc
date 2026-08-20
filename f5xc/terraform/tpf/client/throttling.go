// Package client
// Copyright (c) 2026 F5 Inc. All rights reserved.

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultResourceKey = "unknown"

// ThrottlingConfig is provider-facing configuration for backend-driven request throttling.
type ThrottlingConfig struct {
	MaxRetries        int
	InitialBackoff    string
	MaxBackoff        string
	Multiplier        float64
	JitterRatio       float64
	RespectRetryAfter bool
}

type throttlingPolicy struct {
	Enabled           bool
	MaxRetries        int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	Multiplier        float64
	JitterRatio       float64
	RespectRetryAfter bool
}

// RateLimitCoordinator stores the next time a request is allowed for a key.
type RateLimitCoordinator struct {
	nextAllowedAt atomic.Int64
}

// ResourceThrottleRegistry tracks per-resource cooldown coordinators.
type ResourceThrottleRegistry struct {
	coordinators sync.Map // map[string]*RateLimitCoordinator
}

type ctxResourceKey struct{}

func defaultThrottlingPolicy() throttlingPolicy {
	return throttlingPolicy{
		Enabled:           false,
		MaxRetries:        5,
		InitialBackoff:    250 * time.Millisecond,
		MaxBackoff:        8 * time.Second,
		Multiplier:        2.0,
		JitterRatio:       0.2,
		RespectRetryAfter: true,
	}
}

func resolveThrottlingPolicy(cfg *ThrottlingConfig) (throttlingPolicy, error) {
	p := defaultThrottlingPolicy()
	if cfg == nil {
		return p, nil
	}

	p.Enabled = true
	p.MaxRetries = cfg.MaxRetries
	if cfg.InitialBackoff != "" {
		d, err := time.ParseDuration(cfg.InitialBackoff)
		if err != nil {
			return p, fmt.Errorf("invalid throttling.initial_backoff %q: %w", cfg.InitialBackoff, err)
		}
		p.InitialBackoff = d
	}
	if cfg.MaxBackoff != "" {
		d, err := time.ParseDuration(cfg.MaxBackoff)
		if err != nil {
			return p, fmt.Errorf("invalid throttling.max_backoff %q: %w", cfg.MaxBackoff, err)
		}
		p.MaxBackoff = d
	}
	if cfg.Multiplier > 0 {
		p.Multiplier = cfg.Multiplier
	}
	if cfg.JitterRatio >= 0 {
		p.JitterRatio = cfg.JitterRatio
	}
	p.RespectRetryAfter = cfg.RespectRetryAfter

	if p.MaxRetries < 0 {
		return p, fmt.Errorf("throttling.max_retries must be >= 0")
	}
	if p.InitialBackoff <= 0 {
		return p, fmt.Errorf("throttling.initial_backoff must be > 0")
	}
	if p.MaxBackoff <= 0 {
		return p, fmt.Errorf("throttling.max_backoff must be > 0")
	}
	if p.MaxBackoff < p.InitialBackoff {
		return p, fmt.Errorf("throttling.max_backoff must be >= throttling.initial_backoff")
	}
	if p.Multiplier < 1.0 {
		return p, fmt.Errorf("throttling.multiplier must be >= 1.0")
	}
	if p.JitterRatio < 0 || p.JitterRatio > 1 {
		return p, fmt.Errorf("throttling.jitter_ratio must be in [0,1]")
	}

	return p, nil
}

func newResourceThrottleRegistry() *ResourceThrottleRegistry {
	return &ResourceThrottleRegistry{}
}

func contextFromResourceKey(ctx context.Context, resourceKey string) context.Context {
	return context.WithValue(ctx, ctxResourceKey{}, normalizeResourceKey(resourceKey))
}

func resourceKeyFromContext(ctx context.Context) string {
	resourceKey, _ := ctx.Value(ctxResourceKey{}).(string)
	return normalizeResourceKey(resourceKey)
}

func resourceKeyForCustomRPC(rpcFQN string) string {
	rpcFQN = strings.TrimSpace(rpcFQN)
	if rpcFQN == "" {
		return ""
	}
	parts := strings.Split(rpcFQN, ".")
	if len(parts) <= 1 {
		return normalizeResourceKey(rpcFQN)
	}
	return normalizeResourceKey(strings.Join(parts[:len(parts)-1], "."))
}

func normalizeResourceKey(resourceKey string) string {
	resourceKey = strings.TrimSpace(strings.ToLower(resourceKey))
	if resourceKey == "" {
		return defaultResourceKey
	}
	return resourceKey
}

func deriveResourceKeyFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultResourceKey
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for idx, part := range parts {
		if part == "namespaces" && idx+2 < len(parts) {
			return normalizeResourceKey(parts[idx+2])
		}
	}
	return normalizeResourceKey(parts[len(parts)-1])
}

func resolveRequestResourceKey(req *http.Request) string {
	if req == nil {
		return defaultResourceKey
	}
	if k := resourceKeyFromContext(req.Context()); k != defaultResourceKey {
		return k
	}
	return deriveResourceKeyFromPath(req.URL.Path)
}

func retryDelayForAttempt(attempt int, policy throttlingPolicy, resp *http.Response) time.Duration {
	if policy.RespectRetryAfter {
		d, ok := parseRetryAfterDelay(resp)
		if ok && d > 0 {
			return d
		}
	}
	return computeBackoffDelay(attempt, policy)
}

func (r *ResourceThrottleRegistry) coordinator(resourceKey string) *RateLimitCoordinator {
	resourceKey = normalizeResourceKey(resourceKey)
	coordinator, ok := r.coordinators.Load(resourceKey)
	if ok {
		return coordinator.(*RateLimitCoordinator)
	}
	created := &RateLimitCoordinator{}
	actual, _ := r.coordinators.LoadOrStore(resourceKey, created)
	return actual.(*RateLimitCoordinator)
}

func (r *ResourceThrottleRegistry) wait(ctx context.Context, resourceKey string) error {
	if r == nil {
		return nil
	}
	coordinator := r.coordinator(resourceKey)
	for {
		nextAllowedAt := coordinator.nextAllowedAt.Load()
		if nextAllowedAt == 0 {
			return nil
		}
		remaining := time.Until(time.Unix(0, nextAllowedAt))
		if remaining <= 0 {
			return nil
		}
		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (r *ResourceThrottleRegistry) publish(resourceKey string, delay time.Duration) {
	if r == nil || delay <= 0 {
		return
	}
	coordinator := r.coordinator(resourceKey)
	candidate := time.Now().Add(delay).UnixNano()
	for {
		current := coordinator.nextAllowedAt.Load()
		if current >= candidate {
			return
		}
		if coordinator.nextAllowedAt.CompareAndSwap(current, candidate) {
			return
		}
	}
}

func ensureReplayableBody(req *http.Request) (bool, error) {
	if req == nil || req.Body == nil {
		return true, nil
	}
	if req.GetBody != nil {
		return true, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return false, err
	}
	if errClose := req.Body.Close(); errClose != nil {
		return false, errClose
	}
	rebuild := func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.GetBody = rebuild
	req.Body, _ = rebuild()
	req.ContentLength = int64(len(body))
	return true, nil
}

func cloneRequestForAttempt(req *http.Request) (*http.Request, error) {
	cloned := req.Clone(req.Context())
	if req.GetBody == nil {
		return cloned, nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	cloned.Body = body
	return cloned, nil
}

func parseRetryAfterDelay(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	raw := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if raw == "" {
		return 0, false
	}
	seconds, err := strconv.Atoi(raw)
	if err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	retryAt, err := http.ParseTime(raw)
	if err != nil {
		return 0, false
	}
	delay := time.Until(retryAt)
	if delay < 0 {
		return 0, true
	}
	return delay, true
}

func computeBackoffDelay(attempt int, policy throttlingPolicy) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := float64(policy.InitialBackoff) * math.Pow(policy.Multiplier, float64(attempt-1))
	maxDelay := float64(policy.MaxBackoff)
	if base > maxDelay {
		base = maxDelay
	}
	delay := time.Duration(base)
	if policy.JitterRatio > 0 {
		minFactor := 1.0 - policy.JitterRatio
		maxFactor := 1.0 + policy.JitterRatio
		factor := minFactor + rand.Float64()*(maxFactor-minFactor)
		delay = time.Duration(float64(delay) * factor)
	}
	if delay < 0 {
		return 0
	}
	if delay > policy.MaxBackoff {
		return policy.MaxBackoff
	}
	return delay
}

func closeResponseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func doRequestWithThrottling(
	ctx context.Context,
	baseReq *http.Request,
	resourceKey string,
	policy throttlingPolicy,
	registry *ResourceThrottleRegistry,
	send func(*http.Request) (*http.Response, error),
) (*http.Response, error) {
	if baseReq == nil {
		return nil, fmt.Errorf("nil request")
	}
	if !policy.Enabled || registry == nil {
		return send(baseReq)
	}

	resourceKey = normalizeResourceKey(resourceKey)
	canRetry, err := ensureReplayableBody(baseReq)
	if err != nil {
		return nil, err
	}

	maxAttempts := policy.MaxRetries + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptReq, err := cloneRequestForAttempt(baseReq)
		if err != nil {
			return nil, err
		}
		// Check right before send so a 429 published by any concurrent goroutine
		// stops unsent requests immediately, not just at the next loop iteration.
		if err := registry.wait(ctx, resourceKey); err != nil {
			return nil, err
		}
		resp, err := send(attemptReq)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		if attempt == maxAttempts || !canRetry {
			return resp, nil
		}

		delay := retryDelayForAttempt(attempt, policy, resp)
		closeResponseBody(resp)
		// Publish so fresh goroutines (attempt == 1) respect the backoff window.
		registry.publish(resourceKey, delay)
		// Sleep independently per goroutine so all concurrent retriers don't
		// wake from registry.wait at the same instant (thundering herd).
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, fmt.Errorf("throttling failed to complete request")
}
