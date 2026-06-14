package ratelimit

import (
	"sync"
	"time"
)

// SlidingWindow is a per-key rate limiter that tracks request timestamps within
// a rolling time window. Requests older than the window boundary are evicted
// before each check, so the limit applies to the most recent window duration
// rather than a fixed calendar interval.
type SlidingWindow struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	window     time.Duration
}

// NewSlidingWindow creates a SlidingWindow that allows at most limit requests
// within the given window duration.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{
		timestamps: make([]time.Time, 0, 3),
		limit:      limit,
		window:     window,
	}
}

// Allow reports whether the next request is within the rate limit. It evicts
// timestamps that have fallen outside the window, then records the current
// timestamp and returns true if the count is still within the limit. Returns
// false without recording if the limit has been reached.
func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	cutoff := time.Now().Add(-sw.window)
	for len(sw.timestamps) > 0 && sw.timestamps[0].Before(cutoff) {
		sw.timestamps = sw.timestamps[1:]
	}
	if len(sw.timestamps) >= sw.limit {
		return false
	}

	sw.timestamps = append(sw.timestamps, time.Now())
	return true
}

// Registry manages a SlidingWindow per key, creating one lazily on first use.
// All keys share the same limit and window parameters supplied at construction.
// It is safe for concurrent use.
type Registry struct {
	mu       sync.Mutex
	limiters map[string]*SlidingWindow
	limit    int
	window   time.Duration
}

// NewRegistry creates a Registry with the given limit and window applied to
// every key it manages.
func NewRegistry(limit int, window time.Duration) *Registry {
	return &Registry{
		limiters: make(map[string]*SlidingWindow),
		limit:    limit,
		window:   window,
	}
}

// Allow reports whether the request identified by key is within the rate limit.
// If no limiter exists for key yet, one is created automatically.
func (r *Registry) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	sw, ok := r.limiters[key]
	if !ok {
		sw = NewSlidingWindow(r.limit, r.window)
		r.limiters[key] = sw
	}

	return sw.Allow()
}
