package server

import (
	"errors"
	"net"
	"sync"
)

type Registry struct {
	mu    sync.Mutex
	conns map[net.Conn]bool
}

func NewRegistry() *Registry {
	return &Registry{
		conns: make(map[net.Conn]bool),
	}
}

func (r *Registry) Add(conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[conn] = true

	return nil
}

func (r *Registry) Remove(conn net.Conn) error {
	if conn == nil {
		return errors.New("conn must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, conn)

	return nil
}

func (r *Registry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.conns)
}
