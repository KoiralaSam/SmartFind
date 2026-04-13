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
	"google.golang.org/protobuf/types/known/emptypb"
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

func (h *Handler) CreateLostReport(ctx context.Context, req *pb.CreateLostReportRequest) (*pb.LostReport, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetPassengerId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "passenger_id is required")
	}
	if strings.TrimSpace(req.GetItemName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "item_name is required")
	}

	in := inbound.CreateLostReportInput{
		PassengerID:     req.GetPassengerId(),
		ItemName:        req.GetItemName(),
		ItemDescription: req.GetItemDescription(),
		ItemType:        req.GetItemType(),
		Brand:           req.GetBrand(),
		Model:           req.GetModel(),
		Color:           req.GetColor(),
		Material:        req.GetMaterial(),
		ItemCondition:   req.GetItemCondition(),
		Category:        req.GetCategory(),
		LocationLost:    req.GetLocationLost(),
		RouteOrStation:  req.GetRouteOrStation(),
		RouteID:         req.GetRouteId(),
	}
	if req.GetDateLost() != nil {
		in.DateLost = req.GetDateLost().AsTime()
	}

	report, err := h.usecase.CreateLostReport(ctx, in)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.LostReportToPB(report), nil
}

func (h *Handler) ListLostReports(ctx context.Context, req *pb.ListLostReportsRequest) (*pb.ListLostReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetPassengerId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "passenger_id is required")
	}

	reports, err := h.usecase.ListLostReports(ctx, inbound.ListLostReportsInput{
		PassengerID: req.GetPassengerId(),
		Status:      req.GetStatus(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.ListLostReportsResponse{Reports: mapper.LostReportsToPB(reports)}, nil
}

func (h *Handler) DeleteLostReport(ctx context.Context, req *pb.DeleteLostReportRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetPassengerId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "passenger_id is required")
	}
	if strings.TrimSpace(req.GetLostReportId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "lost_report_id is required")
	}

	if err := h.usecase.DeleteLostReport(ctx, req.GetPassengerId(), req.GetLostReportId()); err != nil {
		return nil, mapDomainError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) SearchFoundItemMatches(ctx context.Context, req *pb.SearchFoundItemMatchesRequest) (*pb.SearchFoundItemMatchesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetPassengerId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "passenger_id is required")
	}
	if strings.TrimSpace(req.GetLostReportId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "lost_report_id is required")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}

	matches, err := h.usecase.SearchFoundItemMatches(ctx, inbound.SearchFoundItemsInput{
		PassengerID:  req.GetPassengerId(),
		LostReportID: req.GetLostReportId(),
		Limit:        limit,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.SearchFoundItemMatchesResponse{Matches: mapper.FoundItemMatchesToPB(matches)}, nil
}

func (h *Handler) FileClaim(ctx context.Context, req *pb.FileClaimRequest) (*pb.ItemClaim, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetPassengerId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "passenger_id is required")
	}
	if strings.TrimSpace(req.GetFoundItemId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "found_item_id is required")
	}
	if strings.TrimSpace(req.GetLostReportId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "lost_report_id is required")
	}

	claim, err := h.usecase.FileClaim(ctx, inbound.FileClaimInput{
		PassengerID:  req.GetPassengerId(),
		FoundItemID:  req.GetFoundItemId(),
		LostReportID: req.GetLostReportId(),
		Message:      req.GetMessage(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.ItemClaimToPB(claim), nil
}

func mapDomainError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not found"):
		return status.Error(codes.NotFound, err.Error())
	case strings.Contains(msg, "permission"), strings.Contains(msg, "forbidden"):
		return status.Error(codes.PermissionDenied, err.Error())
	case strings.Contains(msg, "already exists"), strings.Contains(msg, "duplicate"):
		return status.Error(codes.AlreadyExists, err.Error())
	case strings.Contains(msg, "invalid"), strings.Contains(msg, "required"):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
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
