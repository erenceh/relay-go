package messaging

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/erenceh/relay-go/internal/protocol"
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
	BroadcastRoom(roomName string, msg Message) error
	// DirectMessage sends a private message to a specific online user.
	// Returns an error if the recipient is offline.
	DirectMessage(to string, msg Message) error
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
	mu    sync.Mutex
	rooms map[string]Room     // room name -> room
	users map[string]net.Conn // username -> connection, used for DM routing
}

// NewInMemoryMessageRouter returns an initialized InMemoryMessageRouter.
func NewInMemoryMessageRouter() *InMemoryMessageRouter {
	return &InMemoryMessageRouter{
		rooms: make(map[string]Room),
		users: make(map[string]net.Conn),
	}
}

// Message represents a chat message with a sender and body.
type Message struct {
	From string
	Body string
}

// NewMessage returns an initialized Message with the given sender and body.
func NewMessage(from string, body string) Message {
	return Message{
		From: from,
		Body: body,
	}
}

// Room represents a chat room with a name and its currently connected members.
type Room struct {
	Name    string
	Members map[string]net.Conn
}

// NewRoom returns an initialized Room with the given name.
func NewRoom(name string) Room {
	return Room{
		Name:    name,
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
	room, ok := mr.rooms[roomName]
	if !ok {
		room = NewRoom(roomName)
		mr.rooms[roomName] = room
	}
	room.Members[username] = conn
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
func (mr *InMemoryMessageRouter) BroadcastRoom(roomName string, msg Message) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	room, ok := mr.rooms[roomName]
	if !ok {
		return fmt.Errorf("the room:%s does not exist", roomName)
	}

	broadcastMsg := fmt.Sprintf("[%s] %s: %s", room.Name, msg.From, msg.Body)
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

// DirectMessage sends a private message to the named user.
// Returns an error if the recipient is not currently online.
func (mr *InMemoryMessageRouter) DirectMessage(to string, msg Message) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	user, ok := mr.users[to]
	if !ok {
		return fmt.Errorf("the user:%s is offline", to)
	}

	directMsg := fmt.Sprintf("[DM] %s: %s", msg.From, msg.Body)
	if err := protocol.WriteMessage(user, []byte(directMsg)); err != nil {
		slog.Warn("failed to deliver message", "user", to, "err", err)
	}

	return nil
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
		rooms = append(rooms, room.Name)
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
		response := fmt.Sprintf("%s - members: %s", room.Name, strings.Join(members, ", "))
		if err := protocol.WriteMessage(conn, []byte(response)); err != nil {
			slog.Warn("failed to print room info", "room", room, "err", err)
		}
	}

	return nil
}
