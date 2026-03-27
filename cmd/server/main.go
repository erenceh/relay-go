package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/erenceh/relay-go/internal/auth"
	"github.com/erenceh/relay-go/internal/messaging"
	"github.com/erenceh/relay-go/internal/presence"
	"github.com/erenceh/relay-go/internal/protocol"
	"github.com/erenceh/relay-go/internal/server"
)

func main() {
	network := "tcp"
	address := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	secret := os.Getenv("JWT_SECRET")
	// Temporary secret for testing
	if secret == "" {
		slog.Warn("JWT_SECRET not set, using insecure default")
		secret = "dev-secret-change-me"
	}

	// --- Listener setup ---
	listener, err := net.Listen(network, *address)
	if err != nil {
		slog.Error("failed to start listener", "err", err)
		os.Exit(1)
	}
	defer listener.Close()
	slog.Info("server listening", "addr", *address)

	// --- In-memory state ---
	registry := server.NewRegistry()
	authService := auth.NewInMemoryAuthService([]byte(secret))
	presenceStore := presence.NewInMemoryPresenceStore()
	router := messaging.NewInMemoryMessageRouter()

	// --- Accept loop: spawn a goroutine per client ---
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Warn("accept error", "err", err)
			continue
		}
		if err := registry.Add(conn); err != nil {
			slog.Warn("connection rejected", "addr", conn.RemoteAddr(), "err", err)
			conn.Close()
			continue
		}

		slog.Info("client connected", "addr", conn.RemoteAddr())
		go handleConn(conn, registry, authService, presenceStore, router)
	}
}

// handleConn manages the full lifecycle of a single client connection.
// It runs in its own goroutine for each connected client.
func handleConn(
	conn net.Conn,
	registry *server.Registry,
	authService auth.AuthService,
	presenceStore presence.PresenceStore,
	router messaging.MessageRouter,
) {
	// --- Cleanup on disconnect ---
	defer conn.Close()
	defer registry.Remove(conn)
	defer presenceStore.Remove(conn)
	defer slog.Info("client disconnected", "addr", conn.RemoteAddr())

	// --- Username handshake ---
	protocol.WriteMessage(conn, []byte("welcome to relay-go. /register or /login:"))
	username, err := runAuthLoop(conn, authService)
	if err != nil {
		protocol.WriteMessage(conn, []byte(err.Error()))
		return
	}

	presenceStore.Add(username, conn)
	defer router.Disconnect(username)

	// --- Message loop ---
	runCommandLoop(conn, presenceStore, router, username)
}

func runAuthLoop(conn net.Conn, authService auth.AuthService) (username string, err error) {
authLoop:
	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			return "", fmt.Errorf("connection lost during auth: %w", err)
		}

		userRes := strings.TrimSpace(string(frame.Data))
		switch userRes {
		case "/register":
			protocol.WriteMessage(conn, []byte("enter your username:"))
			userNameFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", fmt.Errorf("connection lost during auth: %w", err)
			}
			username = string(userNameFrame.Data)

			protocol.WriteMessage(conn, []byte("enter your password:"))
			userPasswordFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", fmt.Errorf("connection lost during auth: %w", err)
			}
			password := string(userPasswordFrame.Data)

			if err := authService.Register(username, password); err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue authLoop
			}
			accessToken, refreshToken, err := authService.Login(username, password)
			if err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue authLoop
			}
			username, err = authService.Validate(accessToken)
			if err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue authLoop
			}
			protocol.WriteMessage(conn, []byte("registration successful"))
			protocol.WriteMessage(conn, []byte("refresh:"+refreshToken))
			break authLoop

		case "/login":
			protocol.WriteMessage(conn, []byte("enter your username:"))
			userNameFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", fmt.Errorf("connection lost during auth: %w", err)
			}
			username = string(userNameFrame.Data)

			protocol.WriteMessage(conn, []byte("enter your password:"))
			userPasswordFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", fmt.Errorf("connection lost during auth: %w", err)
			}
			password := string(userPasswordFrame.Data)

			accessToken, refreshToken, err := authService.Login(username, password)
			if err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue authLoop
			}
			username, err = authService.Validate(accessToken)
			if err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue authLoop
			}
			protocol.WriteMessage(conn, []byte("login successful"))
			protocol.WriteMessage(conn, []byte("refresh:"+refreshToken))
			break authLoop

		case "/refresh":
			protocol.WriteMessage(conn, []byte("enter your refresh token:"))
			refreshTokenFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", fmt.Errorf("connection lost during auth: %w", err)
			}
			refreshTokenOld := string(refreshTokenFrame.Data)
			accessToken, refreshToken, err := authService.Refresh(refreshTokenOld)
			if err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue authLoop
			}
			username, err = authService.Validate(accessToken)
			if err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue authLoop
			}
			protocol.WriteMessage(conn, []byte("login successful"))
			protocol.WriteMessage(conn, []byte("refresh:"+refreshToken))
			break authLoop

		default:
			protocol.WriteMessage(conn, []byte("invalid input. please enter /register or /login:"))
			continue
		}
	}

	return username, nil
}

func runCommandLoop(
	conn net.Conn,
	presenceStore presence.PresenceStore,
	router messaging.MessageRouter,
	username string,
) {
	// --- Auto-join default room ---
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

		// --- Command router ---
		switch fields[0] {
		case "/help":
			// TODO improve help formatting when terminal UI library is added (v5)
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
			notification := messaging.NewMessage("server", username+" left the room")
			router.BroadcastRoom(currentRoom, notification)
			router.JoinRoom(defaultRoom, username, conn)
			currentRoom = defaultRoom

		case "/dm":
			if len(fields) < 3 {
				protocol.WriteMessage(conn, []byte("usage: /dm <user> <message>"))
				continue
			}
			body := strings.Join(fields[2:], " ")
			router.DirectMessage(fields[1], messaging.NewMessage(username, body))

		case "/rooms":
			router.PrintRooms(conn)

		case "/who":
			users := presenceStore.List()
			response := "online: " + strings.Join(users, ", ")
			protocol.WriteMessage(conn, []byte(response))

		default:
			// Broadcast regular messages to the user's current room
			if err := router.BroadcastRoom(currentRoom, messaging.NewMessage(username, msg)); err != nil {
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
