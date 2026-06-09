package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"net"
	"os"

	pb "github.com/erenceh/relay-go/gen/proto/presence"
	"github.com/erenceh/relay-go/internal/domain"
	"github.com/erenceh/relay-go/internal/events"
	"github.com/erenceh/relay-go/internal/presence"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	godotenv.Load()

	network := "tcp"
	address := flag.String("addr", ":50053", "listen address")
	flag.Parse()

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

	// -- Listener setup ---
	listener, err := net.Listen(network, *address)
	if err != nil {
		slog.Error("failed to start listener", "err", err)
		os.Exit(1)
	}
	defer listener.Close()
	slog.Info("presence server listening", "addr", *address)

	// -- Create gRPC Presence Server ---
	grpcService := grpc.NewServer()
	presenceService := presence.NewOnlineStore()
	pb.RegisterPresenceServiceServer(grpcService, &presenceGRPCServer{onlineStore: presenceService})
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcService, healthServer)
	healthServer.SetServingStatus("presence", grpc_health_v1.HealthCheckResponse_SERVING)

	nc.Subscribe(events.SubjectUserPresence, func(msg *nats.Msg) {
		var event domain.UserPresenceEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			slog.Warn("failed to deserialize presence event", "err", err)
			return
		}
		presenceService.SetOnline(event.Username, event.IsOnline)
		slog.Info("presence updated", "username", event.Username, "online", event.IsOnline)
	})

	if err := grpcService.Serve(listener); err != nil {
		slog.Error("failed to serve", "err", err)
		os.Exit(1)
	}
}
