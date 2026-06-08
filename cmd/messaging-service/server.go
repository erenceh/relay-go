package main

import (
	"context"
	"fmt"
	"log/slog"

	pb "github.com/erenceh/relay-go/gen/proto/messaging"
	"github.com/erenceh/relay-go/internal/messaging"
)

type messagingGRPCServer struct {
	pb.UnimplementedMessagingServiceServer
	roomStore messaging.RoomStore
}

func (m *messagingGRPCServer) JoinRoom(ctx context.Context, req *pb.JoinRoomRequest) (*pb.JoinRoomResponse, error) {
	slog.Info("user joining room", "username", req.Username, "room", req.RoomName)
	if err := m.roomStore.JoinRoom(req.RoomName, req.Username); err != nil {
		return nil, fmt.Errorf("error joining room: %w", err)
	}
	return &pb.JoinRoomResponse{Success: true}, nil
}

func (m *messagingGRPCServer) LeaveRoom(ctx context.Context, req *pb.LeaveRoomRequest) (*pb.LeaveRoomResponse, error) {
	slog.Info("user leaving room", "username", req.Username, "room", req.RoomName)
	if err := m.roomStore.LeaveRoom(req.RoomName, req.Username); err != nil {
		return nil, fmt.Errorf("error leaving room: %w", err)
	}
	return &pb.LeaveRoomResponse{Success: true}, nil
}

func (m *messagingGRPCServer) ListRooms(ctx context.Context, req *pb.ListRoomsRequest) (*pb.ListRoomsResponse, error) {
	return &pb.ListRoomsResponse{Rooms: m.roomStore.ListRooms()}, nil
}

func (m *messagingGRPCServer) ListRoomMembers(ctx context.Context, req *pb.ListRoomMembersRequest) (*pb.ListRoomMembersResponse, error) {
	roomMembers, err := m.roomStore.ListRoomMembers(req.RoomName)
	if err != nil {
		return nil, fmt.Errorf("error getting room members: %w", err)
	}
	return &pb.ListRoomMembersResponse{RoomMembers: roomMembers}, nil
}

func (m *messagingGRPCServer) GetRoomID(ctx context.Context, req *pb.GetRoomIDRequest) (*pb.GetRoomIDResponse, error) {
	return &pb.GetRoomIDResponse{RoomID: m.roomStore.GetRoomID(req.RoomName)}, nil
}
