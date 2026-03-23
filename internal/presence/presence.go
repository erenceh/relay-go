package presence

import (
	"errors"
	"net"
	"sync"
)

// PresenceStore defines the interface for tracking online users.
// Implementations must be safe for concurrent use.
type PresenceStore interface {
	// Add registers a username and their connection as online.
	Add(username string, conn net.Conn) error
	// Remove marks a connection as offline by searching for it
	// across all registered users.
	Remove(conn net.Conn) error
	// List returns usernames of all currently online users.
	List() []string
}

// InMemoryPresenceStore is an in-memory implementation of PresenceStore.
// It maps usernames to their active TCP connections.
type InMemoryPresenceStore struct {
	mu    sync.Mutex
	conns map[string]net.Conn // username -> active connection
}

// NewInMemoryPresenceStore returns an initialized InMemoryPresenceStore.
func NewInMemoryPresenceStore() *InMemoryPresenceStore {
	return &InMemoryPresenceStore{
		conns: make(map[string]net.Conn),
	}
}

// Add registers the given username and connection as online.
// Returns an error if conn is nil.
func (ps *InMemoryPresenceStore) Add(username string, conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.conns[username] = conn

	return nil
}

// Remove finds and removes the connection from the presence store.
// Searches by conneciton value since the username may not be known
// at disconnect time. Returns an error if conn is nil.
func (ps *InMemoryPresenceStore) Remove(conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Search by value, callers know the connection but not
	// necessarily the username when a client disconnects.
	for username, c := range ps.conns {
		if c == conn {
			delete(ps.conns, username)
		}
	}

	return nil
}

// List returns the usernames of all currently online users.
// Returns a copy of the internal state to avoid exposing the live map.
func (ps *InMemoryPresenceStore) List() []string {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	users := make([]string, 0, len(ps.conns))
	for username := range ps.conns {
		users = append(users, username)
	}

	return users
}
