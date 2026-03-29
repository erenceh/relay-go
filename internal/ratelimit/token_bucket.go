package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm. Tokens accumulate at a
// fixed refill rate up to a maximum capacity. Each allowed request consumes
// one token; requests are denied when the bucket is empty. Unused capacity
// carries over, allowing short bursts up to maxTokens.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

// NewTokenBucket creates a full TokenBucket with the given maximum capacity and
// refill rate in tokens per second.
func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow reports whether a request can proceed. It first refills the bucket
// based on the time elapsed since the last call, then consumes one token and
// returns true. Returns false without consuming a token if the bucket is empty.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	elapsed := time.Since(tb.lastRefill)
	tb.tokens += elapsed.Seconds() * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = time.Now()

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// BucketRegistry manages a TokenBucket per key, creating one lazily on first
// use. All buckets share the same maxTokens and refillRate supplied at
// construction. It is safe for concurrent use.
type BucketRegistry struct {
	mu         sync.Mutex
	buckets    map[string]*TokenBucket
	maxTokens  float64
	refillRate float64
}

// NewBucketReistry creates a BucketRegistry with the given maxTokens capacity
// and refillRate applied to every bucket it manages.
func NewBucketReistry(maxTokens, refillRate float64) *BucketRegistry {
	return &BucketRegistry{
		buckets:    make(map[string]*TokenBucket),
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}

// Allow reports whether the request identified by key is within the rate limit.
// If no bucket exists for key yet, one is created automatically.
func (r *BucketRegistry) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, ok := r.buckets[key]
	if !ok {
		bucket = NewTokenBucket(r.maxTokens, r.refillRate)
		r.buckets[key] = bucket
	}

	return bucket.Allow()
}
