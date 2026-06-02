package main

import (
	"flag"
	"log/slog"
	"net"
	"os"

	pb "github.com/erenceh/relay-go/gen/proto"
	"github.com/erenceh/relay-go/internal/auth"
	"github.com/erenceh/relay-go/internal/db"
	"github.com/erenceh/relay-go/internal/repository"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	godotenv.Load()

	network := "tcp"
	address := flag.String("addr", ":50051", "listen address")
	flag.Parse()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		slog.Warn("JWT_SECRET not set, using insecure default")
		secret = "dev-secret-change-me"
	}

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

	// --- Listener setup ---
	listener, err := net.Listen(network, *address)
	if err != nil {
		slog.Error("failed to start listener", "err", err)
		os.Exit(1)
	}
	defer listener.Close()
	slog.Info("server listening", "addr", *address)

	// --- Create gRPC server ---
	userRepo := repository.NewPostgresUserRepository(database)
	authService := auth.NewAuthService(userRepo, []byte(secret))
	grpcService := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcService, &authGRPCServer{authService: authService})
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcService, healthServer)
	healthServer.SetServingStatus("auth", grpc_health_v1.HealthCheckResponse_SERVING)

	if err := grpcService.Serve(listener); err != nil {
		slog.Error("failed to serve", "err", err)
		os.Exit(1)
	}
}
