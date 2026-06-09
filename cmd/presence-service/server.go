package main

import (
	"context"

	pb "github.com/erenceh/relay-go/gen/proto/presence"
	"github.com/erenceh/relay-go/internal/presence"
)

// presenceGRPCServer implements the gRPC PresenceService, delegating state
// queries to an OnlineStore.
type presenceGRPCServer struct {
	pb.UnimplementedPresenceServiceServer
	onlineStore presence.OnlineStore
}

// ListOnline returns the usernames of all currently online users.
func (p *presenceGRPCServer) ListOnline(ctx context.Context, req *pb.ListOnlineRequest) (*pb.ListOnlineResponse, error) {
	return &pb.ListOnlineResponse{Users: p.onlineStore.ListOnline()}, nil
}
