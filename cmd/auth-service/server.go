package main

import (
	"context"
	"fmt"

	pb "github.com/erenceh/relay-go/gen/proto"
	"github.com/erenceh/relay-go/internal/auth"
)

type authGRPCServer struct {
	pb.UnimplementedAuthServiceServer
	authService auth.AuthService
}

func (a *authGRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if err := a.authService.Register(req.Username, req.Password); err != nil {
		return nil, fmt.Errorf("error registering user: %w", err)
	}

	accessToken, _, err := a.authService.Login(req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("error creating tokens: %w", err)
	}

	_, userID, err := a.authService.Validate(accessToken)
	if err != nil {
		return nil, fmt.Errorf("error validating user: %w", err)
	}

	return &pb.RegisterResponse{
		Username: req.Username,
		UserId:   userID,
		Token:    accessToken,
	}, nil
}

func (a *authGRPCServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	accessToken, _, err := a.authService.Login(req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("error logging in user: %w", err)
	}

	_, userID, err := a.authService.Validate(accessToken)
	if err != nil {
		return nil, fmt.Errorf("error validating user: %w", err)
	}

	return &pb.LoginResponse{
		Username: req.Username,
		UserId:   userID,
		Token:    accessToken,
	}, nil
}

func (a *authGRPCServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	username, _, err := a.authService.Validate(req.Token)
	if err != nil {
		return nil, fmt.Errorf("error validating user: %w", err)
	}

	return &pb.ValidateTokenResponse{Username: username}, nil
}

func (a *authGRPCServer) Refresh(ctx context.Context, req *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	accessToken, refreshToken, err := a.authService.Refresh(req.Token)
	if err != nil {
		return nil, fmt.Errorf("error refreshing token: %w", err)
	}

	username, userID, err := a.authService.Validate(accessToken)
	if err != nil {
		return nil, fmt.Errorf("error validating user: %w", err)
	}

	return &pb.RefreshResponse{
		Username:     username,
		UserId:       userID,
		Token:        accessToken,
		RefreshToken: refreshToken,
	}, nil
}
