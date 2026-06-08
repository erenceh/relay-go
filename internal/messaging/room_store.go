package messaging

import (
	"fmt"
	"sync"

	"github.com/erenceh/relay-go/internal/domain"
	"github.com/erenceh/relay-go/internal/repository"
)

type InMemoryRoomStore struct {
	mu       sync.Mutex
	rooms    map[string]*roomEntry
	roomRepo repository.RoomRepository
}

func NewRoomStore(roomRepo repository.RoomRepository) *InMemoryRoomStore {
	return &InMemoryRoomStore{
		rooms:    make(map[string]*roomEntry),
		roomRepo: roomRepo,
	}
}

type roomEntry struct {
	id      string
	members map[string]struct{}
}

func newRoomEntry(id string) *roomEntry {
	return &roomEntry{
		id:      id,
		members: make(map[string]struct{}),
	}
}

func (rs *InMemoryRoomStore) JoinRoom(roomName string, username string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Create the room if it doesn't exist yet.
	roomSession, ok := rs.rooms[roomName]
	if !ok {
		existingRoom, err := rs.roomRepo.FindByRoomName(roomName)
		if err != nil {
			return fmt.Errorf("failed to find room: %w", err)
		}

		var room domain.Room
		if existingRoom == nil {
			room = domain.NewRoom(roomName)
			if err := rs.roomRepo.Create(&room); err != nil {
				return fmt.Errorf("failed to create room: %w", err)
			}
		} else {
			room = *existingRoom
		}

		roomSession = newRoomEntry(room.ID)
		rs.rooms[roomName] = roomSession
	}

	roomSession.members[username] = struct{}{}

	return nil
}

func (rs *InMemoryRoomStore) LeaveRoom(roomName string, username string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomName]
	if !ok {
		return fmt.Errorf("the room:%s does not exist", roomName)
	}
	delete(room.members, username)

	// Clean up empty rooms to avoid accumulating ghost rooms in memory.
	if len(room.members) == 0 {
		delete(rs.rooms, roomName)
	}

	return nil
}

func (rs *InMemoryRoomStore) ListRooms() []string {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rooms := make([]string, 0, len(rs.rooms))
	for roomName := range rs.rooms {
		rooms = append(rooms, roomName)
	}

	return rooms
}

func (rs *InMemoryRoomStore) ListRoomMembers(roomName string) ([]string, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomName]
	if !ok {
		return []string{}, fmt.Errorf("the room:%s does not exist", roomName)
	}

	members := make([]string, 0, len(room.members))
	for member := range room.members {
		members = append(members, member)
	}

	return members, nil
}

func (rs *InMemoryRoomStore) GetRoomID(roomName string) string {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	session, ok := rs.rooms[roomName]
	if !ok {
		return ""
	}
	return session.id
}
