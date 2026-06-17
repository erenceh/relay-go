package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	authpb "github.com/erenceh/relay-go/backend/gen/proto/auth"
	messagingpb "github.com/erenceh/relay-go/backend/gen/proto/messaging"
	presencepb "github.com/erenceh/relay-go/backend/gen/proto/presence"
	"github.com/erenceh/relay-go/backend/internal/db"
	"github.com/erenceh/relay-go/backend/internal/domain"
	"github.com/erenceh/relay-go/backend/internal/events"
	"github.com/erenceh/relay-go/backend/internal/messaging"
	"github.com/erenceh/relay-go/backend/internal/presence"
	"github.com/erenceh/relay-go/backend/internal/protocol"
	"github.com/erenceh/relay-go/backend/internal/ratelimit"
	"github.com/erenceh/relay-go/backend/internal/repository"
	"github.com/erenceh/relay-go/backend/internal/server"
	wsadapter "github.com/erenceh/relay-go/backend/internal/transport/websocket"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	godotenv.Load()

	network := "tcp"
	address := flag.String("addr", ":8080", "listen address")
	flag.Parse()

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
	authServiceAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authServiceAddr == "" {
		slog.Error("AUTH_SERVICE_ADDR is required")
		os.Exit(1)
	}

	grpcAuthConn, err := grpc.NewClient(authServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to auth service", "err", err)
		os.Exit(1)
	}
	defer grpcAuthConn.Close()

	// --- Messaging Service setup ---
	messagingServiceAddr := os.Getenv("MESSAGING_SERVICE_ADDR")
	if messagingServiceAddr == "" {
		slog.Error("MESSAGING_SERVICE_ADDR is required")
		os.Exit(1)
	}

	grpcMessagingConn, err := grpc.NewClient(messagingServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to messaging service", "err", err)
		os.Exit(1)
	}
	defer grpcMessagingConn.Close()

	// --- Presence Service setup ---
	presenceServiceAddr := os.Getenv("PRESENCE_SERVICE_ADDR")
	if presenceServiceAddr == "" {
		slog.Error("PRESENCE_SERVICE_ADDR is required")
		os.Exit(1)
	}

	grpcPresenceConn, err := grpc.NewClient(presenceServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to messaging service", "err", err)
		os.Exit(1)
	}
	defer grpcPresenceConn.Close()

	// --- NATS setup ---
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		slog.Error("NATS_URL is required")
		os.Exit(1)
	}

	nc, err := events.Connect(natsURL)
	if err != nil {
		slog.Error("failed to connect to NATS", "err", err)
		os.Exit(1)
	}
	defer nc.Close()

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
	authClient := authpb.NewAuthServiceClient(grpcAuthConn)
	messagingClient := messagingpb.NewMessagingServiceClient(grpcMessagingConn)
	presenceClient := presencepb.NewPresenceServiceClient(grpcPresenceConn)
	registrationLimiter := ratelimit.NewRegistry(3, time.Hour)
	messageLimiter := ratelimit.NewBucketReistry(5, 0.5)
	presenceStore := presence.NewInMemoryPresenceStore()
	roomRepo := repository.NewPostgresRoomRepository(database)
	router := messaging.NewInMemoryMessageRouter(roomRepo)
	messageRepo := repository.NewPostgresMessageRepository(database)

	// --- WebSocket Endpoint ---
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		var preAuthUsername, preAuthUserID string

		if token != "" {
			var tokenRes *authpb.ValidateTokenResponse
			err = callWithRetry(func() error {
				var callErr error
				tokenRes, callErr = authClient.ValidateToken(context.Background(), &authpb.ValidateTokenRequest{
					Token: token,
				})
				return callErr
			}, 3, time.Second)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			preAuthUsername = tokenRes.Username
			preAuthUserID = tokenRes.UserId
		}

		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Warn("failed to upgrade websocket connection", "err", err)
			return
		}

		adaptedConn := wsadapter.NewAdapter(wsConn)

		if err := registry.Add(adaptedConn); err != nil {
			slog.Warn("connection rejected", "err", err)
			adaptedConn.Close()
			return
		}

		slog.Info("websocket client connected", "addr", adaptedConn.RemoteAddr())
		go handleConn(
			adaptedConn,
			nc,
			registry,
			authClient,
			messagingClient,
			presenceClient,
			registrationLimiter,
			messageLimiter,
			presenceStore,
			router,
			messageRepo,
			preAuthUsername,
			preAuthUserID,
		)
	})

	// --- HTTP server ---
	go func() {
		slog.Info("websocket server listerning", "addr", ":8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			slog.Error("websocket server failed", "err", err)
		}
	}()

	// --- TCP Accept loop: spawn a goroutine per client ---
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
			nc,
			registry,
			authClient,
			messagingClient,
			presenceClient,
			registrationLimiter,
			messageLimiter,
			presenceStore,
			router,
			messageRepo,
			"", "", // not pre-authenticated
		)
	}
}

// handleConn manages the full lifecycle of a single client connection.
// It runs in its own goroutine for each connected client.
func handleConn(
	conn net.Conn,
	nc *nats.Conn,
	registry *server.Registry,
	authClient authpb.AuthServiceClient,
	messageClient messagingpb.MessagingServiceClient,
	presenceClient presencepb.PresenceServiceClient,
	registrationLimiter *ratelimit.Registry,
	messageLimiter *ratelimit.BucketRegistry,
	presenceStore presence.PresenceStore,
	router messaging.MessageRouter,
	messageRepo repository.MessageRepository,
	preAuthUsername string,
	preAuthUserID string,
) {
	// --- Cleanup on disconnect ---
	defer conn.Close()
	defer registry.Remove(conn)
	defer presenceStore.Remove(conn)
	defer slog.Info("client disconnected", "addr", conn.RemoteAddr())

	conn.SetDeadline(time.Now().Add(2 * time.Minute))

	var username, userID string
	if preAuthUsername != "" {
		username, userID = preAuthUsername, preAuthUserID
	} else {
		protocol.WriteMessage(conn, []byte("welcome to relay-go. /register or /login:"))

		var err error
		username, userID, err = runAuthLoop(conn, authClient, registrationLimiter, presenceStore)
		if err != nil {
			protocol.WriteMessage(conn, []byte(err.Error()))
			return
		}
	}
	conn.SetDeadline(time.Now().Add(5 * time.Minute))

	presenceStore.Add(username, conn)
	defer router.Disconnect(username)

	// --- Message loop ---
	runCommandLoop(
		conn,
		nc,
		messageClient,
		presenceClient,
		messageLimiter,
		presenceStore,
		router,
		username,
		userID,
		messageRepo,
	)
}

func runAuthLoop(
	conn net.Conn,
	authClient authpb.AuthServiceClient,
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

			var registerRes *authpb.RegisterResponse
			err = callWithRetry(func() error {
				var callErr error
				registerRes, callErr = authClient.Register(context.Background(), &authpb.RegisterRequest{
					Username: username,
					Password: password,
				})
				return callErr
			}, 3, time.Second)
			if err != nil {
				sendError("auth service unavailable, please try again later")
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

			var loginRes *authpb.LoginResponse
			err = callWithRetry(func() error {
				var callErr error
				loginRes, callErr = authClient.Login(context.Background(), &authpb.LoginRequest{
					Username: username,
					Password: password,
				})
				return callErr
			}, 3, time.Second)
			if err != nil {
				sendError("auth service unavailable, please try again later")
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

			var refreshRes *authpb.RefreshResponse
			err = callWithRetry(func() error {
				var callErr error
				refreshRes, callErr = authClient.Refresh(context.Background(), &authpb.RefreshRequest{
					Token: refreshTokenOld,
				})
				return callErr
			}, 3, time.Second)
			if err != nil {
				sendError("auth service unavailable, please try again later")
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
	nc *nats.Conn,
	messageClient messagingpb.MessagingServiceClient,
	presenceClient presencepb.PresenceServiceClient,
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
	messageClient.JoinRoom(context.Background(), &messagingpb.JoinRoomRequest{
		RoomName: currentRoom,
		Username: username,
	})
	router.JoinRoom(currentRoom, username, conn)
	if err := events.Publish(nc, events.SubjectUserJoined, domain.UserJoinedEvent{
		Username: username,
		RoomName: currentRoom,
	}); err != nil {
		slog.Warn("failed to publish user joined event", "username", username, "roomName", currentRoom, "err", err)
	}
	if err := events.Publish(nc, events.SubjectUserPresence, domain.UserPresenceEvent{
		Username: username,
		IsOnline: true,
	}); err != nil {
		slog.Warn("failed to publish user online event", "username", username, "err", err)
	}

	roomID, err := messageClient.GetRoomID(context.Background(), &messagingpb.GetRoomIDRequest{RoomName: currentRoom})
	if err != nil {
		slog.Warn("failed to get room id", "roomName", currentRoom, "err", err)
	}

	messages, err := messageRepo.ListByRoom(roomID.RoomID, 20)
	if err != nil {
		slog.Warn("failed to load message history", "room", currentRoom, "err", err)
	} else {
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			formatted := fmt.Sprintf("[%s] %s: %s", currentRoom, msg.From, msg.Body)
			protocol.WriteMessage(conn, []byte(formatted))
		}
	}

	// --- Command loop ---
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
			messageClient.LeaveRoom(context.Background(), &messagingpb.LeaveRoomRequest{
				RoomName: currentRoom,
				Username: username,
			})
			messageClient.JoinRoom(context.Background(), &messagingpb.JoinRoomRequest{
				RoomName: roomName,
				Username: username,
			})
			router.JoinRoom(roomName, username, conn)
			currentRoom = roomName

			roomID, err := messageClient.GetRoomID(context.Background(), &messagingpb.GetRoomIDRequest{RoomName: currentRoom})
			if err != nil {
				slog.Warn("failed to get room id", "roomName", currentRoom, "err", err)
			}
			messages, err := messageRepo.ListByRoom(roomID.RoomID, 20)
			if err != nil {
				slog.Warn("failed to load message history", "room", currentRoom, "err", err)
			} else {
				for i := len(messages) - 1; i >= 0; i-- {
					msg := messages[i]
					formatted := fmt.Sprintf("[%s] %s: %s", currentRoom, msg.From, msg.Body)
					protocol.WriteMessage(conn, []byte(formatted))
				}
			}

			if err := events.Publish(nc, events.SubjectUserJoined, domain.UserJoinedEvent{
				Username: username,
				RoomName: currentRoom,
			}); err != nil {
				slog.Warn("failed to publish user joined event", "username", username, "roomName", currentRoom, "err", err)
			}

			notification := messaging.NewMessage("server", username+" joined the room")
			router.BroadcastRoom(currentRoom, notification)

		case "/leave":
			messageClient.LeaveRoom(context.Background(), &messagingpb.LeaveRoomRequest{
				RoomName: currentRoom,
				Username: username,
			})

			if err := events.Publish(nc, events.SubjectUserLeft, domain.UserLeftEvent{
				Username: username,
				RoomName: currentRoom,
			}); err != nil {
				slog.Warn("failed to publish user left event", "username", username, "roomName", currentRoom, "err", err)
			}

			notification := messaging.NewMessage("server", username+" left the room")
			router.BroadcastRoom(currentRoom, notification)

			messageClient.JoinRoom(context.Background(), &messagingpb.JoinRoomRequest{
				RoomName: defaultRoom,
				Username: username,
			})
			router.JoinRoom(defaultRoom, username, conn)
			currentRoom = defaultRoom

			if err := events.Publish(nc, events.SubjectUserJoined, domain.UserJoinedEvent{
				Username: username,
				RoomName: currentRoom,
			}); err != nil {
				slog.Warn("failed to publish user joined event", "username", username, "roomName", currentRoom, "err", err)
			}

			notification = messaging.NewMessage("server", username+" joined the room")
			router.BroadcastRoom(currentRoom, notification)

		case "/rooms":
			router.PrintRooms(conn)

		case "/who":
			users, err := presenceClient.ListOnline(context.Background(), &presencepb.ListOnlineRequest{})
			if err != nil {
				slog.Warn("failed to get online users", "err", err)
			}
			response := "online: " + strings.Join(users.Users, ", ")
			protocol.WriteMessage(conn, []byte(response))

		default:
			if !messageLimiter.Allow(username) {
				protocol.WriteMessage(conn, []byte("rate limit exceeded"))
				continue
			}

			if err := router.BroadcastRoom(currentRoom, messaging.NewMessage(username, msg)); err != nil {
				protocol.WriteMessage(conn, []byte("you must be in a room to send messages"))
			}

			roomID, err := messageClient.GetRoomID(context.Background(), &messagingpb.GetRoomIDRequest{RoomName: currentRoom})
			if err != nil {
				slog.Warn("failed to get room id", "roomName", currentRoom, "err", err)
			}
			domainMsg := domain.NewMessage(userID, roomID.RoomID, username, msg)
			if err := messageRepo.Create(&domainMsg); err != nil {
				slog.Warn("failed to persist message", "err", err)
			}

			if err := events.Publish(nc, events.SubjectMessageSent, domain.MessageSentEvent{
				Sender:   username,
				RoomName: currentRoom,
				Message:  msg,
			}); err != nil {
				slog.Warn("failed to publish message event", "username", username, "roomName", currentRoom, "err", err)
			}

			slog.Info("message received",
				"addr", conn.RemoteAddr(),
				"user", username,
				"msg", string(frame.Data),
			)
		}
	}

	// --- User disconnected ---
	if err := events.Publish(nc, events.SubjectUserPresence, domain.UserPresenceEvent{
		Username: username,
		IsOnline: false,
	}); err != nil {
		slog.Warn("failed to publish user offline event", "username", username, "err", err)
	}
}

func callWithRetry(fn func() error, maxAttempts int, initialDelay time.Duration) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err != nil {
			slog.Warn("auth service call failed", "attempt", attempt, "err", err)
			time.Sleep(initialDelay)
			initialDelay *= 2
			if initialDelay > 30*time.Second {
				initialDelay = 30 * time.Second
			}
			continue
		}
		return nil
	}
	slog.Error("failed to connect")
	return errors.New("max retry attempts reached")
}
