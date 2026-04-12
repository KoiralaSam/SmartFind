package handler

import (
	"context"
	"errors"
	"strings"

	"smartfind/services/passenger-service/internal/adapters/primary/grpc/mapper"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	pb "smartfind/shared/proto/passenger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedPassengerServiceServer
	usecase inbound.PassengerUsecase
}

func New(usecase inbound.PassengerUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	idToken := strings.TrimSpace(req.GetIdToken())
	if idToken == "" {
		return nil, status.Error(codes.InvalidArgument, "id_token is required")
	}

	res, err := h.usecase.Login(ctx, inbound.LoginInput{IDToken: idToken})
	if err != nil {
		return nil, mapLoginError(err)
	}
	if res == nil {
		return nil, status.Error(codes.Internal, "login failed")
	}

	return &pb.LoginResponse{
		Passenger:    mapper.PassengerToPB(res.Passenger),
		SessionToken: res.SessionToken,
	}, nil
}

func mapLoginError(err error) error {
	if err == nil {
		return status.Error(codes.Internal, "unknown error")
	}

	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "id_token is required"):
		return status.Error(codes.InvalidArgument, "id_token is required")
	case strings.Contains(msg, "google_client_id is required"):
		return status.Error(codes.Internal, "server not configured")
	case strings.Contains(msg, "google tokeninfo"):
		return status.Error(codes.Unauthenticated, "invalid id_token")
	case strings.Contains(msg, "audience"):
		return status.Error(codes.Unauthenticated, "invalid id_token")
	case strings.Contains(msg, "not verified"):
		return status.Error(codes.Unauthenticated, "invalid id_token")
	default:
		return status.Error(codes.Internal, "login failed")
	}
}
