package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/erenceh/relay-go/gen/proto"
	"github.com/erenceh/relay-go/internal/db"
	"github.com/erenceh/relay-go/internal/domain"
	"github.com/erenceh/relay-go/internal/messaging"
	"github.com/erenceh/relay-go/internal/presence"
	"github.com/erenceh/relay-go/internal/protocol"
	"github.com/erenceh/relay-go/internal/ratelimit"
	"github.com/erenceh/relay-go/internal/repository"
	"github.com/erenceh/relay-go/internal/server"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	godotenv.Load()

	network := "tcp"
	address := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	authServiceAddr := os.Getenv("AUTH_SERVICE_ADDR")
	// Temporary secret for testing
	if authServiceAddr == "" {
		slog.Error("AUTH_SERVICE_ADDR is required")
		os.Exit(1)
	}

	// --- Database setup ---
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "db/migrations"
	}

	if err := db.RunMigrations(database, migrationsPath); err != nil {
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected and migrations applied")

	retentionDays := 30
	if val := os.Getenv("RETENTION_DAYS"); val != "" {
		if days, err := strconv.Atoi(val); err == nil {
			retentionDays = days
		}
	}
	retention := time.Duration(retentionDays) * 24 * time.Hour
	db.StartCleanupWorker(database, retention)
	slog.Info("cleanup worker started", "retention_days", retentionDays)

	// --- Auth Service setup ---
	grpcConn, err := grpc.NewClient(authServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to auth service", "err", err)
		os.Exit(1)
	}
	defer grpcConn.Close()

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
	authClient := pb.NewAuthServiceClient(grpcConn)
	registrationLimiter := ratelimit.NewRegistry(3, time.Hour)
	messageLimiter := ratelimit.NewBucketReistry(5, 0.5)
	presenceStore := presence.NewInMemoryPresenceStore()
	roomRepo := repository.NewPostgresRoomRepository(database)
	router := messaging.NewInMemoryMessageRouter(roomRepo)
	messageRepo := repository.NewPostgresMessageRepository(database)

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
		go handleConn(
			conn,
			registry,
			authClient,
			registrationLimiter,
			messageLimiter,
			presenceStore,
			router,
			messageRepo)
	}
}

// handleConn manages the full lifecycle of a single client connection.
// It runs in its own goroutine for each connected client.
func handleConn(
	conn net.Conn,
	registry *server.Registry,
	authClient pb.AuthServiceClient,
	registrationLimiter *ratelimit.Registry,
	messageLimiter *ratelimit.BucketRegistry,
	presenceStore presence.PresenceStore,
	router messaging.MessageRouter,
	messageRepo repository.MessageRepository,
) {
	// --- Cleanup on disconnect ---
	defer conn.Close()
	defer registry.Remove(conn)
	defer presenceStore.Remove(conn)
	defer slog.Info("client disconnected", "addr", conn.RemoteAddr())

	conn.SetDeadline(time.Now().Add(2 * time.Minute))

	// --- Username handshake ---
	protocol.WriteMessage(conn, []byte("welcome to relay-go. /register or /login:"))
	username, userID, err := runAuthLoop(conn, authClient, registrationLimiter, presenceStore)
	if err != nil {
		protocol.WriteMessage(conn, []byte(err.Error()))
		return
	}
	conn.SetDeadline(time.Now().Add(5 * time.Minute))

	presenceStore.Add(username, conn)
	defer router.Disconnect(username)

	// --- Message loop ---
	runCommandLoop(conn, messageLimiter, presenceStore, router, username, userID, messageRepo)
}

func runAuthLoop(
	conn net.Conn,
	authClient pb.AuthServiceClient,
	registrationLimiter *ratelimit.Registry,
	presenceStore presence.PresenceStore,
) (username, userID string, err error) {
	sendError := func(msg string) {
		protocol.WriteMessage(conn, []byte(msg))
		protocol.WriteMessage(conn, []byte("enter /register or /login:"))
	}

authLoop:
	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			return "", "", fmt.Errorf("connection lost during auth: %w", err)
		}

		userRes := strings.TrimSpace(string(frame.Data))
		switch userRes {
		case "/register":
			addr := conn.RemoteAddr().String()
			host, _, _ := net.SplitHostPort(addr)
			if !registrationLimiter.Allow(host) {
				sendError("registration rate limit exceeded - try again later")
				continue authLoop
			}

			protocol.WriteMessage(conn, []byte("enter your username:"))
			userNameFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", "", fmt.Errorf("connection lost during auth: %w", err)
			}
			username = string(userNameFrame.Data)

			protocol.WriteMessage(conn, []byte("enter your password:"))
			userPasswordFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", "", fmt.Errorf("connection lost during auth: %w", err)
			}
			password := string(userPasswordFrame.Data)

			registerRes, err := authClient.Register(context.Background(), &pb.RegisterRequest{
				Username: username,
				Password: password,
			})
			if err != nil {
				sendError(err.Error())
				continue authLoop
			}

			username = registerRes.Username
			userID = registerRes.UserId

			protocol.WriteMessage(conn, []byte("registration successful"))
			protocol.WriteMessage(conn, []byte("refresh:"+registerRes.Token))
			break authLoop

		case "/login":
			protocol.WriteMessage(conn, []byte("enter your username:"))
			userNameFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", "", fmt.Errorf("connection lost during auth: %w", err)
			}
			username = string(userNameFrame.Data)

			protocol.WriteMessage(conn, []byte("enter your password:"))
			userPasswordFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", "", fmt.Errorf("connection lost during auth: %w", err)
			}
			password := string(userPasswordFrame.Data)

			loginRes, err := authClient.Login(context.Background(), &pb.LoginRequest{
				Username: username,
				Password: password,
			})
			if err != nil {
				sendError(err.Error())
				continue authLoop
			}

			for _, u := range presenceStore.List() {
				if u == loginRes.Username {
					sendError("user already logged in")
					continue authLoop
				}
			}

			username = loginRes.Username
			userID = loginRes.UserId

			protocol.WriteMessage(conn, []byte("login successful"))
			protocol.WriteMessage(conn, []byte("refresh:"+loginRes.Token))
			break authLoop

		case "/refresh":
			protocol.WriteMessage(conn, []byte("enter your refresh token:"))
			refreshTokenFrame, err := protocol.ReadMessage(conn)
			if err != nil {
				return "", "", fmt.Errorf("connection lost during auth: %w", err)
			}
			refreshTokenOld := string(refreshTokenFrame.Data)
			refreshRes, err := authClient.Refresh(context.Background(), &pb.RefreshRequest{
				Token: refreshTokenOld,
			})
			if err != nil {
				sendError(err.Error())
				continue authLoop
			}

			username = refreshRes.Username
			userID = refreshRes.UserId

			protocol.WriteMessage(conn, []byte("login successful"))
			protocol.WriteMessage(conn, []byte("refresh:"+refreshRes.RefreshToken))
			break authLoop

		default:
			protocol.WriteMessage(conn, []byte("invalid input. please enter /register or /login:"))
			continue
		}
	}

	return username, userID, nil
}

func runCommandLoop(
	conn net.Conn,
	messageLimiter *ratelimit.BucketRegistry,
	presenceStore presence.PresenceStore,
	router messaging.MessageRouter,
	username string,
	userID string,
	messageRepo repository.MessageRepository,
) {
	// --- Auto-join default room ---
	defaultRoom := "general chat"
	currentRoom := defaultRoom
	router.JoinRoom(currentRoom, username, conn)

	roomID := router.GetRoomID(currentRoom)
	messages, err := messageRepo.ListByRoom(roomID, 20)
	if err != nil {
		slog.Warn("failed to load message history", "room", currentRoom, "err", err)
	} else {
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			formatted := fmt.Sprintf("[%s] %s: %s", currentRoom, msg.From, msg.Body)
			protocol.WriteMessage(conn, []byte(formatted))
		}
	}

	for {
		frame, err := protocol.ReadMessage(conn)
		if err != nil {
			break
		}
		conn.SetDeadline(time.Now().Add(5 * time.Minute))

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
			if err := domain.ValidateRoomName(roomName); err != nil {
				protocol.WriteMessage(conn, []byte(err.Error()))
				continue
			}
			router.LeaveRoom(currentRoom, username)
			router.JoinRoom(roomName, username, conn)
			currentRoom = roomName

			roomID := router.GetRoomID(currentRoom)
			messages, err := messageRepo.ListByRoom(roomID, 20)
			if err != nil {
				slog.Warn("failed to load message history", "room", currentRoom, "err", err)
			} else {
				for i := len(messages) - 1; i >= 0; i-- {
					msg := messages[i]
					formatted := fmt.Sprintf("[%s] %s: %s", currentRoom, msg.From, msg.Body)
					protocol.WriteMessage(conn, []byte(formatted))
				}
			}

			notification := messaging.NewMessage("server", username+" joined the room")
			router.BroadcastRoom(currentRoom, notification)

		case "/leave":
			router.LeaveRoom(currentRoom, username)
			notification := messaging.NewMessage("server", username+" left the room")
			router.BroadcastRoom(currentRoom, notification)
			router.JoinRoom(defaultRoom, username, conn)
			currentRoom = defaultRoom

		case "/rooms":
			router.PrintRooms(conn)

		case "/who":
			users := presenceStore.List()
			response := "online: " + strings.Join(users, ", ")
			protocol.WriteMessage(conn, []byte(response))

		default:
			if !messageLimiter.Allow(username) {
				protocol.WriteMessage(conn, []byte("rate limit exceeded"))
				continue
			}

			if err := router.BroadcastRoom(currentRoom, messaging.NewMessage(username, msg)); err != nil {
				protocol.WriteMessage(conn, []byte("you must be in a room to send messages"))
			}

			roomID := router.GetRoomID(currentRoom)
			domainMsg := domain.NewMessage(userID, roomID, username, msg)
			if err := messageRepo.Create(&domainMsg); err != nil {
				slog.Warn("failed to persist message", "err", err)
			}

			slog.Info("message received",
				"addr", conn.RemoteAddr(),
				"user", username,
				"msg", string(frame.Data),
			)
		}
	}
}
