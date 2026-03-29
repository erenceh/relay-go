package messaging

import (
	"net"
	"testing"

	"github.com/erenceh/relay-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoomRepo struct {
	rooms map[string]*domain.Room
}

func newMockRoomRepo() *mockRoomRepo {
	return &mockRoomRepo{rooms: make(map[string]*domain.Room)}
}

func (m *mockRoomRepo) Create(room *domain.Room) error {
	m.rooms[room.Name] = room
	return nil
}

func (m *mockRoomRepo) FindByRoomName(name string) (*domain.Room, error) {
	room, ok := m.rooms[name]
	if !ok {
		return nil, nil
	}
	return room, nil
}

func (m *mockRoomRepo) FindByRoomID(id string) (*domain.Room, error) {
	for _, room := range m.rooms {
		if room.ID == id {
			return room, nil
		}
	}
	return nil, nil
}

func TestMessaging(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn)
	}{
		{
			name: "JoinRoom adds user to existing room",
			run: func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn) {
				require.NoError(t, router.JoinRoom("lobby", "alice", conn))
				require.NoError(t, router.JoinRoom("lobby", "bob", conn))

				members, err := router.ListRoomMembers("lobby")
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"alice", "bob"}, members)
			},
		},
		{
			name: "JoinRoom creates room if it does not exist",
			run: func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn) {
				require.NoError(t, router.JoinRoom("newroom", "alice", conn))
				assert.Contains(t, router.ListRooms(), "newroom")
			},
		},
		{
			name: "LeaveRoom removes user from room",
			run: func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn) {
				require.NoError(t, router.JoinRoom("lobby", "alice", conn))
				require.NoError(t, router.JoinRoom("lobby", "bob", conn))

				require.NoError(t, router.LeaveRoom("lobby", "alice"))

				members, err := router.ListRoomMembers("lobby")
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"bob"}, members)
			},
		},
		{
			name: "LeaveRoom deletes empty room",
			run: func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn) {
				require.NoError(t, router.JoinRoom("lobby", "alice", conn))
				require.NoError(t, router.LeaveRoom("lobby", "alice"))

				assert.NotContains(t, router.ListRooms(), "lobby")
			},
		},
		{
			name: "BroadcastRoom returns error for nonexistent room",
			run: func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn) {
				err := router.BroadcastRoom("ghost", NewMessage("alice", "hello"))
				assert.Error(t, err)
			},
		},
		{
			name: "Disconnect removes user from all rooms",
			run: func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn) {
				require.NoError(t, router.JoinRoom("lobby", "alice", conn))
				require.NoError(t, router.JoinRoom("general", "alice", conn))

				router.Disconnect("alice")

				assert.NotContains(t, router.ListRooms(), "lobby")
				assert.NotContains(t, router.ListRooms(), "general")
			},
		},
		{
			name: "ListRooms returns correct room names",
			run: func(t *testing.T, router *InMemoryMessageRouter, conn net.Conn) {
				require.NoError(t, router.JoinRoom("lobby", "alice", conn))
				require.NoError(t, router.JoinRoom("general", "bob", conn))

				assert.ElementsMatch(t, []string{"lobby", "general"}, router.ListRooms())
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := NewInMemoryMessageRouter(newMockRoomRepo())

			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()
			conn := server

			tt.run(t, router, conn)
		})
	}
}
