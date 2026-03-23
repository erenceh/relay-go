package main

import (
	"log/slog"
	"net"
	"strings"

	"github.com/erenceh/relay-go/internal/presence"
	"github.com/erenceh/relay-go/internal/protocol"
	"github.com/erenceh/relay-go/internal/server"
)

func main() {
	network := "tcp"
	address := ":8080"

	listener, err := net.Listen(network, address)
	if err != nil {
		slog.Error("failed to start listener", "err", err)
	}
	defer listener.Close()

	slog.Info("server listening", "addr", address)

	registry := server.NewRegistry()
	presenceStore := presence.NewInMemoryPresenceStore()

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Warn("accept error", "err", err)
			continue
		}
		registry.Add(conn)
		slog.Info("client connected", "addr", conn.RemoteAddr())
		go handleConn(conn, registry, presenceStore)
	}
}

func handleConn(conn net.Conn, registry *server.Registry, presenceStore *presence.InMemoryPresenceStore) {
	defer conn.Close()
	defer registry.Remove(conn)
	defer presenceStore.Remove(conn)
	defer slog.Info("client disconnected", "addr", conn.RemoteAddr())

	// send prompt to client
	protocol.WriteMessage(conn, []byte("enter username:"))
	// read username from client
	frame, err := protocol.ReadMessage(conn)
	if err != nil {
		return
	}
	username := string(frame.Data)

	presenceStore.Add(username, conn)

	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			break
		}

		msg := string(frame.Data)
		if msg == "/who" {
			users := presenceStore.List()
			response := "online: " + strings.Join(users, ", ")
			protocol.WriteMessage(conn, []byte(response))
		} else {
			slog.Info("message received",
				"addr", conn.RemoteAddr(),
				"user", username,
				"msg", string(frame.Data),
			)
		}
	}
}
