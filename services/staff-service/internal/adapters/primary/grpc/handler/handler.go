package handler

import (
	"context"
	"errors"
	"strings"

	"smartfind/services/staff-service/internal/adapters/primary/grpc/mapper"
	"smartfind/services/staff-service/internal/core/ports/inbound"
	"smartfind/services/staff-service/internal/core/ports/outbound"
	pb "smartfind/shared/proto/staff"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	pb.UnimplementedStaffServiceServer
	usecase inbound.StaffUsecase
}

func New(usecase inbound.StaffUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	email := strings.TrimSpace(req.GetEmail())
	password := strings.TrimSpace(req.GetPassword())
	if email == "" || password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	res, err := h.usecase.Login(ctx, inbound.LoginInput{Email: email, Password: password})
	if err != nil {
		return nil, mapLoginError(err)
	}
	if res == nil {
		return nil, status.Error(codes.Internal, "login failed")
	}

	return &pb.LoginResponse{
		Staff:        mapper.StaffToPB(res.Staff),
		SessionToken: res.SessionToken,
	}, nil
}

func (h *Handler) CreateStaff(ctx context.Context, req *pb.CreateStaffRequest) (*pb.Staff, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetTransitCode()) == "" {
		return nil, status.Error(codes.InvalidArgument, "transit_code is required")
	}
	if strings.TrimSpace(req.GetEmail()) == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if strings.TrimSpace(req.GetPassword()) == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	st, err := h.usecase.CreateStaff(ctx, inbound.CreateStaffInput{
		TransitCode: req.GetTransitCode(),
		FullName:    req.GetFullName(),
		Email:       req.GetEmail(),
		Password:    req.GetPassword(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.StaffToPB(st), nil
}

func (h *Handler) CreateFoundItem(ctx context.Context, req *pb.CreateFoundItemRequest) (*pb.FoundItem, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetStaffId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "staff_id is required")
	}
	if strings.TrimSpace(req.GetItemName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "item_name is required")
	}

	in := inbound.CreateFoundItemInput{
		StaffID:         req.GetStaffId(),
		ItemName:        req.GetItemName(),
		ItemDescription: req.GetItemDescription(),
		ItemType:        req.GetItemType(),
		Brand:           req.GetBrand(),
		Model:           req.GetModel(),
		Color:           req.GetColor(),
		Material:        req.GetMaterial(),
		ItemCondition:   req.GetItemCondition(),
		Category:        req.GetCategory(),
		LocationFound:   req.GetLocationFound(),
		RouteOrStation:  req.GetRouteOrStation(),
		RouteID:         req.GetRouteId(),
	}
	if req.GetDateFound() != nil {
		in.DateFound = req.GetDateFound().AsTime()
	}

	it, err := h.usecase.CreateFoundItem(ctx, in)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.FoundItemToPB(it), nil
}

func (h *Handler) UpdateFoundItemStatus(ctx context.Context, req *pb.UpdateFoundItemStatusRequest) (*pb.FoundItem, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetStaffId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "staff_id is required")
	}
	if strings.TrimSpace(req.GetFoundItemId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "found_item_id is required")
	}
	if strings.TrimSpace(req.GetStatus()) == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	it, err := h.usecase.UpdateFoundItemStatus(ctx, inbound.UpdateFoundItemStatusInput{
		StaffID:     req.GetStaffId(),
		FoundItemID: req.GetFoundItemId(),
		Status:      req.GetStatus(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.FoundItemToPB(it), nil
}

func (h *Handler) ListFoundItems(ctx context.Context, req *pb.ListFoundItemsRequest) (*pb.ListFoundItemsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	items, err := h.usecase.ListFoundItems(ctx, inbound.ListFoundItemsInput{
		Status:          req.GetStatus(),
		RouteID:         req.GetRouteId(),
		PostedByStaffID: req.GetPostedByStaffId(),
		Limit:           int(req.GetLimit()),
		Offset:          int(req.GetOffset()),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.ListFoundItemsResponse{Items: mapper.FoundItemsToPB(items)}, nil
}

func (h *Handler) ListClaims(ctx context.Context, req *pb.ListClaimsRequest) (*pb.ListClaimsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := h.usecase.ListClaims(ctx, inbound.ListClaimsInput{
		Status:      req.GetStatus(),
		ItemID:      req.GetItemId(),
		PassengerID: req.GetPassengerId(),
		Limit:       int(req.GetLimit()),
		Offset:      int(req.GetOffset()),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.ListClaimsResponse{Claims: mapper.ItemClaimsToPB(claims)}, nil
}

func (h *Handler) ReviewClaim(ctx context.Context, req *pb.ReviewClaimRequest) (*pb.ItemClaim, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetStaffId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "staff_id is required")
	}
	if strings.TrimSpace(req.GetClaimId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "claim_id is required")
	}
	if strings.TrimSpace(req.GetDecision()) == "" {
		return nil, status.Error(codes.InvalidArgument, "decision is required")
	}

	claim, err := h.usecase.ReviewClaim(ctx, inbound.ReviewClaimInput{
		StaffID:  req.GetStaffId(),
		ClaimID:  req.GetClaimId(),
		Decision: req.GetDecision(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.ItemClaimToPB(claim), nil
}

func (h *Handler) CreateRoute(ctx context.Context, req *pb.CreateRouteRequest) (*pb.Route, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetStaffId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "staff_id is required")
	}
	if strings.TrimSpace(req.GetRouteName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "route_name is required")
	}

	rt, err := h.usecase.CreateRoute(ctx, inbound.CreateRouteInput{
		StaffID:   req.GetStaffId(),
		RouteName: req.GetRouteName(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.RouteToPB(rt), nil
}

func (h *Handler) DeleteRoute(ctx context.Context, req *pb.DeleteRouteRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.GetStaffId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "staff_id is required")
	}
	if strings.TrimSpace(req.GetRouteId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "route_id is required")
	}

	if err := h.usecase.DeleteRoute(ctx, inbound.DeleteRouteInput{
		StaffID: req.GetStaffId(),
		RouteID: req.GetRouteId(),
	}); err != nil {
		return nil, mapDomainError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) ListRoutes(ctx context.Context, req *pb.ListRoutesRequest) (*pb.ListRoutesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	routes, err := h.usecase.ListRoutes(ctx, inbound.ListRoutesInput{
		CreatedByStaffID: req.GetCreatedByStaffId(),
		Limit:            int(req.GetLimit()),
		Offset:           int(req.GetOffset()),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.ListRoutesResponse{Routes: mapper.RoutesToPB(routes)}, nil
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
	if errors.Is(err, outbound.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, outbound.ErrStaffEmailExists) || errors.Is(err, outbound.ErrRouteNameExists) {
		return status.Error(codes.AlreadyExists, err.Error())
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "not found"):
		return status.Error(codes.NotFound, err.Error())
	case strings.Contains(msg, "already exists"), strings.Contains(msg, "duplicate"):
		return status.Error(codes.AlreadyExists, err.Error())
	case strings.Contains(msg, "permission"), strings.Contains(msg, "forbidden"):
		return status.Error(codes.PermissionDenied, err.Error())
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
	case strings.Contains(msg, "email and password are required"):
		return status.Error(codes.InvalidArgument, "email and password are required")
	case strings.Contains(msg, "invalid email or password"):
		return status.Error(codes.Unauthenticated, "invalid email or password")
	case strings.Contains(msg, "missing jwt_secret"):
		return status.Error(codes.Internal, "server not configured")
	default:
		return status.Error(codes.Internal, "login failed")
	}
}
