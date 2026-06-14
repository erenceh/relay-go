package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSlidingWindow(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, sw *SlidingWindow)
	}{
		{
			name: "Allow requests under the limit",
			run: func(t *testing.T, sw *SlidingWindow) {
				assert.True(t, sw.Allow())
				assert.True(t, sw.Allow())
				assert.True(t, sw.Allow())
			},
		},
		{
			name: "Block request at the limit",
			run: func(t *testing.T, sw *SlidingWindow) {
				sw.Allow()
				sw.Allow()
				sw.Allow()
				assert.False(t, sw.Allow())
			},
		},
		{
			name: "Allow again after window expires",
			run: func(t *testing.T, sw *SlidingWindow) {
				sw.Allow()
				sw.Allow()
				sw.Allow()
				assert.False(t, sw.Allow())
				time.Sleep(25 * time.Millisecond)
				assert.True(t, sw.Allow())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := NewSlidingWindow(3, 20*time.Millisecond)
			tt.run(t, sw)
		})
	}
}

func TestRegistry(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, r *Registry)
	}{
		{
			name: "Multiple keys tracked independently",
			run: func(t *testing.T, r *Registry) {
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
			r := NewRegistry(2, time.Minute)
			tt.run(t, r)
		})
	}
}
