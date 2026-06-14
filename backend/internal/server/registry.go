package server

import (
	"errors"
	"net"
	"sync"
)

// Registry tracks all active TCP connections to the server.
// It is safe for concurrent use via a mutex.
type Registry struct {
	mu    sync.Mutex
	conns map[net.Conn]bool
}

// NewRegistry returns an initialized Registry.
func NewRegistry() *Registry {
	return &Registry{
		conns: make(map[net.Conn]bool),
	}
}

// Add registers a connection as active.
// Returns an error if conn is nil.
func (r *Registry) Add(conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.conns) >= 100 {
		return errors.New("server full")
	}

	r.conns[conn] = true

	return nil
}

// Remove deregisters a connection.
// Returns an error if conn is nil.
func (r *Registry) Remove(conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, conn)

	return nil
}

// Count returns the number of currently active connections.
func (r *Registry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.conns)
}
