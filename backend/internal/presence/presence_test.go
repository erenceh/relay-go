package presence

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresence(t *testing.T) {
	for _, tt := range []struct {
		name    string
		run     func(t *testing.T, presenceStore *InMemoryPresenceStore, conn net.Conn)
		nilConn bool
	}{
		{
			name: "Add a user and verify List includes them",
			run: func(t *testing.T, presenceStore *InMemoryPresenceStore, conn net.Conn) {
				err := presenceStore.Add("jim", conn)
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"jim"}, presenceStore.List())
			},
			nilConn: false,
		},
		{
			name: "Add two users and verify both appear in List",
			run: func(t *testing.T, presenceStore *InMemoryPresenceStore, conn net.Conn) {
				server2, client2 := net.Pipe()
				defer server2.Close()
				defer client2.Close()

				err := presenceStore.Add("jim", conn)
				require.NoError(t, err)
				err = presenceStore.Add("bob", server2)
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"jim", "bob"}, presenceStore.List())
			},
			nilConn: false,
		},
		{
			name: "Remove by conection and verify user is no longer in List",
			run: func(t *testing.T, presenceStore *InMemoryPresenceStore, conn net.Conn) {
				err := presenceStore.Add("jim", conn)
				require.NoError(t, err)

				err = presenceStore.Remove(conn)
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{}, presenceStore.List())
			},
			nilConn: false,
		},
		{
			name: "Add nil connection returns error",
			run: func(t *testing.T, presenceStore *InMemoryPresenceStore, conn net.Conn) {
				err := presenceStore.Add("jim", nil)
				assert.Error(t, err)
			},
			nilConn: true,
		},
		{
			name: "Remove nil connection returns error",
			run: func(t *testing.T, presenceStore *InMemoryPresenceStore, conn net.Conn) {
				err := presenceStore.Remove(nil)
				assert.Error(t, err)
			},
			nilConn: true,
		},
		{
			name: "Remove a connection that was never added, should not panic",
			run: func(t *testing.T, presenceStore *InMemoryPresenceStore, conn net.Conn) {
				assert.NotPanics(t, func() {
					presenceStore.Remove(conn)
				})
			},
			nilConn: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			presenceStore := NewInMemoryPresenceStore()

			var conn net.Conn
			if !tt.nilConn {
				server, client := net.Pipe()
				defer server.Close()
				defer client.Close()
				conn = server
			}

			tt.run(t, presenceStore, conn)
		})
	}
}
