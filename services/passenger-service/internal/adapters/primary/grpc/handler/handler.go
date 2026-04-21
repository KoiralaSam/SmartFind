package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"smartfind/services/passenger-service/internal/adapters/primary/grpc/mapper"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/services/passenger-service/internal/core/ports/outbound"
	"smartfind/shared/auth"
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

func requirePassengerClaims(ctx context.Context) (auth.Claims, error) {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return auth.Claims{}, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if claims.ActorType != auth.ActorPassenger || strings.TrimSpace(claims.PassengerID) == "" {
		return auth.Claims{}, status.Error(codes.PermissionDenied, "forbidden")
	}
	return claims, nil
}

func enforcePassengerIDMatch(reqPassengerID string, claims auth.Claims) error {
	reqPassengerID = strings.TrimSpace(reqPassengerID)
	if reqPassengerID != "" && reqPassengerID != claims.PassengerID {
		return status.Error(codes.PermissionDenied, "passenger_id mismatch")
	}
	return nil
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
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforcePassengerIDMatch(req.GetPassengerId(), claims); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetItemName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "item_name is required")
	}

	in := inbound.CreateLostReportInput{
		PassengerID:     claims.PassengerID,
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

// UpdateLostReport patches the passenger's lost report. Each embedded-slot
// string field on the request is forwarded only when non-empty; to
// intentionally clear a slot, callers should send a single space (which
// the repository trims to "") until a dedicated clear semantic is needed.
// date_lost is forwarded only when a timestamp is actually set.
func (h *Handler) UpdateLostReport(ctx context.Context, req *pb.UpdateLostReportRequest) (*pb.LostReport, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforcePassengerIDMatch(req.GetPassengerId(), claims); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetLostReportId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "lost_report_id is required")
	}

	in := inbound.UpdateLostReportInput{
		PassengerID:  claims.PassengerID,
		LostReportID: req.GetLostReportId(),
	}
	strPtrIfSet := func(v string) *string {
		if v == "" {
			return nil
		}
		vv := v
		return &vv
	}
	in.ItemName = strPtrIfSet(req.GetItemName())
	in.ItemDescription = strPtrIfSet(req.GetItemDescription())
	in.ItemType = strPtrIfSet(req.GetItemType())
	in.Brand = strPtrIfSet(req.GetBrand())
	in.Model = strPtrIfSet(req.GetModel())
	in.Color = strPtrIfSet(req.GetColor())
	in.Material = strPtrIfSet(req.GetMaterial())
	in.ItemCondition = strPtrIfSet(req.GetItemCondition())
	in.Category = strPtrIfSet(req.GetCategory())
	in.LocationLost = strPtrIfSet(req.GetLocationLost())
	in.RouteOrStation = strPtrIfSet(req.GetRouteOrStation())
	in.RouteID = strPtrIfSet(req.GetRouteId())
	if req.GetDateLost() != nil {
		t := req.GetDateLost().AsTime()
		in.DateLost = &t
	}

	report, err := h.usecase.UpdateLostReport(ctx, in)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.LostReportToPB(report), nil
}

func (h *Handler) ListLostReports(ctx context.Context, req *pb.ListLostReportsRequest) (*pb.ListLostReportsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforcePassengerIDMatch(req.GetPassengerId(), claims); err != nil {
		return nil, err
	}

	reports, err := h.usecase.ListLostReports(ctx, inbound.ListLostReportsInput{
		PassengerID: claims.PassengerID,
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
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforcePassengerIDMatch(req.GetPassengerId(), claims); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetLostReportId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "lost_report_id is required")
	}

	if err := h.usecase.DeleteLostReport(ctx, claims.PassengerID, req.GetLostReportId()); err != nil {
		return nil, mapDomainError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) SearchFoundItemMatches(ctx context.Context, req *pb.SearchFoundItemMatchesRequest) (*pb.SearchFoundItemMatchesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforcePassengerIDMatch(req.GetPassengerId(), claims); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetLostReportId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "lost_report_id is required")
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}

	matches, err := h.usecase.SearchFoundItemMatches(ctx, inbound.SearchFoundItemsInput{
		PassengerID:  claims.PassengerID,
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
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforcePassengerIDMatch(req.GetPassengerId(), claims); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetFoundItemId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "found_item_id is required")
	}
	if strings.TrimSpace(req.GetLostReportId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "lost_report_id is required")
	}

	claim, err := h.usecase.FileClaim(ctx, inbound.FileClaimInput{
		PassengerID:  claims.PassengerID,
		FoundItemID:  req.GetFoundItemId(),
		LostReportID: req.GetLostReportId(),
		Message:      req.GetMessage(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return mapper.ItemClaimToPB(claim), nil
}

func (h *Handler) ListMyClaims(ctx context.Context, req *pb.ListMyClaimsRequest) (*pb.ListMyClaimsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}

	in := inbound.ListMyClaimsInput{
		PassengerID: claims.PassengerID,
		Status:      req.GetStatus(),
		Limit:       int(req.GetLimit()),
		Offset:      int(req.GetOffset()),
	}

	items, err := h.usecase.ListMyClaims(ctx, in)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.ListMyClaimsResponse{Claims: mapper.ItemClaimsToPB(items)}, nil
}

func (h *Handler) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}

	var createdBefore time.Time
	if req.GetCreatedBefore() != nil {
		createdBefore = req.GetCreatedBefore().AsTime()
	}

	notes, err := h.usecase.ListNotifications(ctx, inbound.ListNotificationsInput{
		PassengerID:   claims.PassengerID,
		Limit:         limit,
		UnreadOnly:    req.GetUnreadOnly(),
		CreatedBefore: createdBefore,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &pb.ListNotificationsResponse{
		Notifications: mapper.PassengerMatchNotificationsToPB(notes),
	}, nil
}

func (h *Handler) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationReadRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requirePassengerClaims(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.usecase.MarkNotificationRead(ctx, inbound.MarkNotificationReadInput{
		PassengerID:     claims.PassengerID,
		NotificationIDs: req.GetNotificationIds(),
	}); err != nil {
		return nil, mapDomainError(err)
	}
	return &emptypb.Empty{}, nil
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
	if errors.Is(err, outbound.ErrLostReportHasActiveClaims) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}
	if errors.Is(err, outbound.ErrLostReportNotFound) {
		return status.Error(codes.NotFound, err.Error())
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
