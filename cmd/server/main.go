package main

import (
	"log/slog"
	"net"

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

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Warn("accept error", "err", err)
			continue
		}
		registry.Add(conn)
		slog.Info("client connected", "addr", conn.RemoteAddr())
		go handleConn(conn, registry)
	}
}

func handleConn(conn net.Conn, registry *server.Registry) {
	defer conn.Close()
	defer registry.Remove(conn)
	defer slog.Info("client disconnected", "addr", conn.RemoteAddr())

	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			break
		}
		slog.Info("message received",
			"addr", conn.RemoteAddr(),
			"msg", string(frame.Data),
		)
	}
}
