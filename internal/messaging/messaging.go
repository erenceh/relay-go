package messaging

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/erenceh/relay-go/internal/domain"
	"github.com/erenceh/relay-go/internal/protocol"
	"github.com/erenceh/relay-go/internal/repository"
)

// MessageRouter defines the interface for routing messages between users and rooms.
// Implementations must be safe for concurrent use.
type MessageRouter interface {
	// JoinRoom adds a user to a named room, creating it if it doesn't exist.
	JoinRoom(roomName string, username string, conn net.Conn) error
	// LeaveRoom removes a user from the named room.
	// Returns an error if the room does not exist.
	LeaveRoom(roomName string, username string) error
	// BroadcastRoom sends a message to all members of the named room.
	// Returns an error if the room does not exists.
	BroadcastRoom(roomName string, msg domain.Message) error
	// SendTo sends a message directly to a named user if they are online.
	SendTo(username string, data []byte) error
	// GetRoomID returns the room id from the given room name.
	GetRoomID(roomName string) string
	// Disconnect removes the user from all rooms and the user map on disconnect.
	Disconnect(username string)
	// ListRooms returns the names of all currently active rooms.
	ListRooms() []string
	// ListRoomMembers returns the names of all current members in the named room.
	// Returns an error if the room does not exists.
	ListRoomMembers(roomName string) ([]string, error)
	// PrintRooms sends a protocol WriteMessage of all currently active rooms and it's members.
	PrintRooms(conn net.Conn) error
}

// InMemoryMessageRouter is an in-memory implementation of MessageRouter.
// It manages active rooms and connected users for message routing.
// All operations are safe for concurrent use via a mutex.
type InMemoryMessageRouter struct {
	mu       sync.Mutex
	rooms    map[string]*RoomSession // room name -> room
	users    map[string]net.Conn     // username -> connection, used for DM routing
	roomRepo repository.RoomRepository
}

// NewInMemoryMessageRouter returns an initialized InMemoryMessageRouter.
func NewInMemoryMessageRouter(roomRepo repository.RoomRepository) *InMemoryMessageRouter {
	return &InMemoryMessageRouter{
		rooms:    make(map[string]*RoomSession),
		users:    make(map[string]net.Conn),
		roomRepo: roomRepo,
	}
}

// RoomSession holds the runtime state for an active room: its domain metadata
// and the set of currently connected members. Members is keyed by username and
// maps to the underlying TCP connection used to write messages to that user.
type RoomSession struct {
	Room    domain.Room
	Members map[string]net.Conn
}

// NewRoomSession returns an initialized RoomSession for the given room with an
// empty Members map ready to accept connections.
func NewRoomSession(room domain.Room) *RoomSession {
	return &RoomSession{
		Room:    room,
		Members: make(map[string]net.Conn),
	}
}

// JoinRoom adds the user to the named room, registering their connection
// for broadcasting. Creates the room if it does not already exist.
// Also registers the user in the router for DM routing.
// Returns an error if conn is nil.
func (mr *InMemoryMessageRouter) JoinRoom(roomName string, username string, conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Create the room if it doesn't exist yet.
	roomSession, ok := mr.rooms[roomName]
	if !ok {
		existingRoom, err := mr.roomRepo.FindByRoomName(roomName)
		if err != nil {
			return fmt.Errorf("failed to find room: %w", err)
		}

		var room domain.Room
		if existingRoom == nil {
			room = domain.NewRoom(roomName)
			if err := mr.roomRepo.Create(&room); err != nil {
				return fmt.Errorf("failed to create room: %w", err)
			}
		} else {
			room = *existingRoom
		}

		roomSession = NewRoomSession(room)
		mr.rooms[roomName] = roomSession
	}
	roomSession.Members[username] = conn
	mr.users[username] = conn

	return nil
}

// LeaveRoom removes the user from the named room.
// If the room becomes empty after removal, it is delated to free memory.
// Returns an error if the room does not exist.
func (mr *InMemoryMessageRouter) LeaveRoom(roomName string, username string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	room, ok := mr.rooms[roomName]
	if !ok {
		return fmt.Errorf("the room:%s does not exist", roomName)
	}
	delete(room.Members, username)

	// Clean up empty rooms to avoid accumulating ghost rooms in memory.
	if len(room.Members) == 0 {
		delete(mr.rooms, roomName)
	}

	return nil
}

// BroadcastRoom sends a formatted message to all members of the named room.
// Failed deliveries are logged and skipped, a bad connection does not
// interrupt delivery to other members.
// Returns an error if the room does not exist.
func (mr *InMemoryMessageRouter) BroadcastRoom(roomName string, msg domain.Message) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	room, ok := mr.rooms[roomName]
	if !ok {
		return fmt.Errorf("the room:%s does not exist", roomName)
	}

	broadcastMsg := fmt.Sprintf("[%s] %s: %s", room.Room.Name, msg.From, msg.Body)
	for username, conn := range room.Members {
		if err := protocol.WriteMessage(conn, []byte(broadcastMsg)); err != nil {
			slog.Warn("failed to deliver message",
				"room", roomName,
				"user", username,
				"err", err,
			)
		}
	}

	return nil
}

// SendTo sends a message directly to a named user if they are online.
func (mr *InMemoryMessageRouter) SendTo(username string, data []byte) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	conn, ok := mr.users[username]
	if !ok {
		return fmt.Errorf("user %s is not online", username)
	}

	return protocol.WriteMessage(conn, data)
}

// GetRoomID returns the room id from the given room name.
func (mr *InMemoryMessageRouter) GetRoomID(roomName string) string {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	session, ok := mr.rooms[roomName]
	if !ok {
		return ""
	}
	return session.Room.ID
}

// Disconnect removes the user from all rooms and the user map.
// Called when a client disconnects.
func (mr *InMemoryMessageRouter) Disconnect(username string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Remove from all rooms they're a member of
	for roomName, room := range mr.rooms {
		delete(room.Members, username)
		// clean up empty rooms
		if len(room.Members) == 0 {
			delete(mr.rooms, roomName)
		}
	}

	// Remove from user map
	delete(mr.users, username)
}

// ListRooms returns the names of all currently active rooms.
// Returns a copy to avoid exposing the internal map.
func (mr *InMemoryMessageRouter) ListRooms() []string {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	rooms := make([]string, 0, len(mr.rooms))
	for _, room := range mr.rooms {
		rooms = append(rooms, room.Room.Name)
	}

	return rooms
}

// ListRoomMembers returns the names of all members in the named room.
// Returns an error if room does not exist
func (mr *InMemoryMessageRouter) ListRoomMembers(roomName string) ([]string, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	room, ok := mr.rooms[roomName]
	if !ok {
		return []string{}, fmt.Errorf("the room:%s does not exist", roomName)
	}

	members := make([]string, 0, len(room.Members))
	for member := range room.Members {
		members = append(members, member)
	}

	return members, nil
}

// PrintRooms sends a protocol WriteMessage of all currently active rooms and it's members.
// Returns an error if conn is nil
func (mr *InMemoryMessageRouter) PrintRooms(conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	mr.mu.Lock()
	defer mr.mu.Unlock()

	if len(mr.rooms) == 0 {
		protocol.WriteMessage(conn, []byte("no active rooms"))
		return nil
	}

	for _, room := range mr.rooms {
		members := make([]string, 0, len(room.Members))
		for member := range room.Members {
			members = append(members, member)
		}
		response := fmt.Sprintf("%s - members: %s", room.Room.Name, strings.Join(members, ", "))
		if err := protocol.WriteMessage(conn, []byte(response)); err != nil {
			slog.Warn("failed to print room info", "room", room, "err", err)
		}
	}

	return nil
}

// NewMessage is a temporary domain.NewMessage wrapper for missing senderID or roomID.
func NewMessage(from, body string) domain.Message {
	return domain.NewMessage("", "", from, body)
}

// NewSystemMessage creates a server notification message.
func NewSystemMessage(roomID, body string) domain.Message {
	return domain.NewMessage(domain.SystemSenderID, roomID, domain.SystemUsername, body)
}

// NewUserMessage creates a message from a real user.
func NewUserMessage(senderID, roomID, from, body string) domain.Message {
	return domain.NewMessage(senderID, roomID, from, body)
}
