package server

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	for _, tt := range []struct {
		name    string
		run     func(t *testing.T, registry *Registry, conn net.Conn)
		nilConn bool
	}{
		{
			name: "verify Count increase when Add connection",
			run: func(t *testing.T, registry *Registry, conn net.Conn) {
				err := registry.Add(conn)
				require.NoError(t, err)
				assert.Equal(t, 1, registry.Count())
			},
		},
		{
			name: "verify Count decrease when Remove connection",
			run: func(t *testing.T, registry *Registry, conn net.Conn) {
				_ = registry.Add(conn)
				err := registry.Remove(conn)
				require.NoError(t, err)
				assert.Equal(t, 0, registry.Count())
			},
		},
		{
			name: "should not panic when Remove a connection that was never added",
			run: func(t *testing.T, registry *Registry, conn net.Conn) {
				assert.NotPanics(t, func() {
					registry.Remove(conn)
				})
			},
		},
		{
			name:    "error when Add a nil connection",
			nilConn: true,
			run: func(t *testing.T, registry *Registry, conn net.Conn) {
				err := registry.Add(nil)
				assert.Error(t, err)
			},
		},
		{
			name:    "error when Remove a nil connection",
			nilConn: true,
			run: func(t *testing.T, registry *Registry, conn net.Conn) {
				err := registry.Remove(nil)
				assert.Error(t, err)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			var conn net.Conn
			if !tt.nilConn {
				server, client := net.Pipe()
				defer server.Close()
				defer client.Close()
				conn = server
			}

			tt.run(t, registry, conn)
		})
	}
}
