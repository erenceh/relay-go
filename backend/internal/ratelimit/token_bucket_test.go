package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucket(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, tb *TokenBucket)
	}{
		{
			name: "Allow requests under the limit",
			run: func(t *testing.T, tb *TokenBucket) {
				assert.True(t, tb.Allow())
				assert.True(t, tb.Allow())
				assert.True(t, tb.Allow())
			},
		},
		{
			name: "Block request when bucket is empty",
			run: func(t *testing.T, tb *TokenBucket) {
				tb.Allow()
				tb.Allow()
				tb.Allow()
				assert.False(t, tb.Allow())
			},
		},
		{
			name: "Allow again after bucket refills",
			run: func(t *testing.T, tb *TokenBucket) {
				tb.Allow()
				tb.Allow()
				tb.Allow()
				assert.False(t, tb.Allow())
				time.Sleep(25 * time.Millisecond)
				assert.True(t, tb.Allow())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 3 max tokens, refill rate of 100 tokens/s (1 token per 10ms)
			tb := NewTokenBucket(3, 100)
			tt.run(t, tb)
		})
	}
}

func TestBucketRegistry(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, r *BucketRegistry)
	}{
		{
			name: "Multiple keys tracked independently",
			run: func(t *testing.T, r *BucketRegistry) {
				assert.True(t, r.Allow("192.168.1.1"))
				assert.True(t, r.Allow("192.168.1.1"))
				assert.False(t, r.Allow("192.168.1.1"))

				assert.True(t, r.Allow("10.0.0.1"))
				assert.True(t, r.Allow("10.0.0.1"))
				assert.False(t, r.Allow("10.0.0.1"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// refillRate 0 disables refilling so the bucket stays empty after exhaustion
			r := NewBucketReistry(2, 0)
			tt.run(t, r)
		})
	}
}
