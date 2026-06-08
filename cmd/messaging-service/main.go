package main

import (
	"flag"
	"log/slog"
	"net"
	"os"

	pb "github.com/erenceh/relay-go/gen/proto/messaging"
	"github.com/erenceh/relay-go/internal/db"
	"github.com/erenceh/relay-go/internal/messaging"
	"github.com/erenceh/relay-go/internal/repository"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	godotenv.Load()

	network := "tcp"
	address := flag.String("addr", ":50052", "listen address")
	flag.Parse()

	// --- Connect to PostreSQL Database ---
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

	// -- Listener setup ---
	listener, err := net.Listen(network, *address)
	if err != nil {
		slog.Error("failed to start listener", "err", err)
		os.Exit(1)
	}
	defer listener.Close()
	slog.Info("messaging server listening", "addr", *address)

	// -- Create gRPC Messaging Server ---
	grpcService := grpc.NewServer()
	messagingRepo := repository.NewPostgresRoomRepository(database)
	messagingService := messaging.NewRoomStore(messagingRepo)
	pb.RegisterMessagingServiceServer(grpcService, &messagingGRPCServer{roomStore: messagingService})
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcService, healthServer)
	healthServer.SetServingStatus("messaging", grpc_health_v1.HealthCheckResponse_SERVING)

	if err := grpcService.Serve(listener); err != nil {
		slog.Error("failed to serve", "err", err)
		os.Exit(1)
	}
}
