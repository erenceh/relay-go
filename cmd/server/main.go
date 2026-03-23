package main

import (
	"log/slog"
	"net"

	"github.com/erenceh/relay-go/internal/server"
)

func main() {
	network := "tcp"
	address := ":8080"
	slog.Info("server listening", "addr", address)

	listener, err := net.Listen(network, address)
	if err != nil {
		slog.Error("failed to start listener", "err", err)
	}
	defer listener.Close()

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
	defer slog.Info("client disconnected", "addr", conn.RemoteAddr())
	defer registry.Remove(conn)

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		conn.Write(buf[:n])
	}
}
