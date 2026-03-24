package main

import (
	"log/slog"
	"net"
	"strings"

	"github.com/erenceh/relay-go/internal/messaging"
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
	router := messaging.NewInMemoryMessageRouter()

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Warn("accept error", "err", err)
			continue
		}
		registry.Add(conn)
		slog.Info("client connected", "addr", conn.RemoteAddr())
		go handleConn(conn, registry, presenceStore, router)
	}
}

func handleConn(
	conn net.Conn,
	registry *server.Registry,
	presenceStore presence.PresenceStore,
	router messaging.MessageRouter,
) {
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
	defer router.Disconnect(username)

	defaultRoom := "general chat"
	currentRoom := defaultRoom
	router.JoinRoom(currentRoom, username, conn)

	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			break
		}

		msg := strings.TrimSpace(string(frame.Data))
		fields := strings.Fields(msg)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "/help":
			// TODO improve help formatting when terminal UI library is added
			commands := `
			/join <room>			Join a room
			/leave					Leave current room
			/dm <user> <msg>		Send a direct message
			/rooms					List active rooms and their members
			/who					List active users
			/quit					Disconnect
			`
			protocol.WriteMessage(conn, []byte(commands))

		case "/join":
			if len(fields) < 2 {
				protocol.WriteMessage(conn, []byte("usage: /join <room>"))
				continue
			}

			roomName := strings.Join(fields[1:], " ")
			router.LeaveRoom(currentRoom, username)
			router.JoinRoom(roomName, username, conn)
			currentRoom = roomName
			notification := messaging.NewMessage("server", username+" joined the room")
			router.BroadcastRoom(currentRoom, notification)

		case "/leave":
			router.LeaveRoom(currentRoom, username)
			router.JoinRoom(defaultRoom, username, conn)
			currentRoom = defaultRoom
			notification := messaging.NewMessage("server", username+" left the room")
			router.BroadcastRoom(currentRoom, notification)

		case "/dm":
			if len(fields) < 3 {
				protocol.WriteMessage(conn, []byte("usage: /dm <user> <message>"))
				continue
			}

			body := strings.Join(fields[2:], " ")
			msg := messaging.NewMessage(username, body)
			router.DirectMessage(fields[1], msg)

		case "/rooms":
			router.PrintRooms(conn)

		case "/who":
			users := presenceStore.List()
			response := "online: " + strings.Join(users, ", ")
			protocol.WriteMessage(conn, []byte(response))

		default:
			message := messaging.NewMessage(username, msg)
			if err := router.BroadcastRoom(currentRoom, message); err != nil {
				protocol.WriteMessage(conn, []byte("you must be in a room to send messages"))
			}
			slog.Info("message received",
				"addr", conn.RemoteAddr(),
				"user", username,
				"msg", string(frame.Data),
			)
		}
	}
}
