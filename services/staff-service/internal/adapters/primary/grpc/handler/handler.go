package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"smartfind/services/staff-service/internal/adapters/primary/grpc/mapper"
	s3media "smartfind/services/staff-service/internal/adapters/secondary/s3"
	"smartfind/services/staff-service/internal/core/ports/inbound"
	"smartfind/services/staff-service/internal/core/ports/outbound"
	"smartfind/shared/auth"
	"smartfind/shared/env"
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

func requireStaffClaims(ctx context.Context) (auth.Claims, error) {
	claims, err := auth.ClaimsFromContext(ctx)
	if err != nil {
		return auth.Claims{}, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if claims.ActorType != auth.ActorStaff || strings.TrimSpace(claims.StaffID) == "" {
		return auth.Claims{}, status.Error(codes.PermissionDenied, "forbidden")
	}
	return claims, nil
}

func enforceStaffIDMatch(reqStaffID string, claims auth.Claims) error {
	reqStaffID = strings.TrimSpace(reqStaffID)
	if reqStaffID == "" {
		return status.Error(codes.InvalidArgument, "staff_id is required")
	}
	if reqStaffID != claims.StaffID {
		return status.Error(codes.PermissionDenied, "staff_id mismatch")
	}
	return nil
}

func attachPresignedImageURLs(ctx context.Context, it *pb.FoundItem) {
	if it == nil {
		return
	}
	keys := it.GetImageKeys()
	if len(keys) == 0 {
		return
	}

	p, err := s3media.GetPresigner(ctx)
	if err != nil || p == nil {
		return
	}

	urls := make([]string, 0, len(keys))
	for _, k := range keys {
		u, err := p.PresignGet(ctx, strings.TrimSpace(k))
		if err != nil || strings.TrimSpace(u) == "" {
			continue
		}
		urls = append(urls, u)
	}
	it.ImageUrls = urls

	primaryKey := strings.TrimSpace(it.GetPrimaryImageKey())
	if primaryKey != "" {
		if u, err := p.PresignGet(ctx, primaryKey); err == nil && strings.TrimSpace(u) != "" {
			it.PrimaryImageUrl = u
			return
		}
	}
	if len(urls) > 0 {
		it.PrimaryImageUrl = urls[0]
	}
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
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforceStaffIDMatch(req.GetStaffId(), claims); err != nil {
		return nil, err
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
		ImageKeys:       req.GetImageKeys(),
		PrimaryImageKey: req.GetPrimaryImageKey(),
	}
	if len(in.ImageKeys) > 5 {
		return nil, status.Error(codes.InvalidArgument, "too many images (max 5)")
	}
	if strings.TrimSpace(in.PrimaryImageKey) == "" && len(in.ImageKeys) > 0 {
		in.PrimaryImageKey = in.ImageKeys[0]
	}
	if req.GetDateFound() != nil {
		in.DateFound = req.GetDateFound().AsTime()
	}

	it, err := h.usecase.CreateFoundItem(ctx, in)
	if err != nil {
		return nil, mapDomainError(err)
	}
	out := mapper.FoundItemToPB(it)
	attachPresignedImageURLs(ctx, out)
	return out, nil
}

func (h *Handler) UpdateFoundItem(ctx context.Context, req *pb.UpdateFoundItemRequest) (*pb.FoundItem, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforceStaffIDMatch(req.GetStaffId(), claims); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetFoundItemId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "found_item_id is required")
	}

	in := inbound.UpdateFoundItemInput{
		StaffID:         req.GetStaffId(),
		FoundItemID:     req.GetFoundItemId(),
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
	if len(req.GetImageKeys()) > 0 {
		keys := req.GetImageKeys()
		in.ImageKeys = &keys
	}
	if req.GetPrimaryImageKey() != "" {
		pk := req.GetPrimaryImageKey()
		in.PrimaryImageKey = &pk
	}

	it, err := h.usecase.UpdateFoundItem(ctx, in)
	if err != nil {
		return nil, mapDomainError(err)
	}
	out := mapper.FoundItemToPB(it)
	attachPresignedImageURLs(ctx, out)
	return out, nil
}

func (h *Handler) DeleteFoundItem(ctx context.Context, req *pb.DeleteFoundItemRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforceStaffIDMatch(req.GetStaffId(), claims); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetFoundItemId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "found_item_id is required")
	}

	if err := h.usecase.DeleteFoundItem(ctx, inbound.DeleteFoundItemInput{
		StaffID:     req.GetStaffId(),
		FoundItemID: req.GetFoundItemId(),
	}); err != nil {
		return nil, mapDomainError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) UpdateFoundItemStatus(ctx context.Context, req *pb.UpdateFoundItemStatusRequest) (*pb.FoundItem, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforceStaffIDMatch(req.GetStaffId(), claims); err != nil {
		return nil, err
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
	out := mapper.FoundItemToPB(it)
	attachPresignedImageURLs(ctx, out)
	return out, nil
}

func (h *Handler) ListFoundItems(ctx context.Context, req *pb.ListFoundItemsRequest) (*pb.ListFoundItemsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}

	postedBy := strings.TrimSpace(req.GetPostedByStaffId())
	if postedBy != "" && postedBy != claims.StaffID {
		return nil, status.Error(codes.PermissionDenied, "posted_by_staff_id mismatch")
	}
	postedBy = claims.StaffID

	items, err := h.usecase.ListFoundItems(ctx, inbound.ListFoundItemsInput{
		Status:          req.GetStatus(),
		RouteID:         req.GetRouteId(),
		PostedByStaffID: postedBy,
		Limit:           int(req.GetLimit()),
		Offset:          int(req.GetOffset()),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	outItems := mapper.FoundItemsToPB(items)
	for _, it := range outItems {
		attachPresignedImageURLs(ctx, it)
	}
	return &pb.ListFoundItemsResponse{Items: outItems}, nil
}

func (h *Handler) InitFoundItemImageUploads(ctx context.Context, req *pb.InitFoundItemImageUploadsRequest) (*pb.InitFoundItemImageUploadsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(claims.ForwardedToken) == "" {
		return nil, status.Error(codes.Unauthenticated, "missing session token")
	}
	if len(req.GetFiles()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "files is required")
	}

	maxSize := int64(env.GetInt("MEDIA_MAX_IMAGE_BYTES", 8*1024*1024))
	if maxSize <= 0 {
		maxSize = 8 * 1024 * 1024
	}

	p, err := s3media.GetPresigner(ctx)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	environment := env.GetString("ENVIRONMENT", "development")
	now := time.Now()

	out := &pb.InitFoundItemImageUploadsResponse{Uploads: make([]*pb.UploadInit, 0, len(req.GetFiles()))}
	for _, f := range req.GetFiles() {
		if f == nil {
			return nil, status.Error(codes.InvalidArgument, "file is required")
		}
		if f.GetSizeBytes() <= 0 {
			return nil, status.Error(codes.InvalidArgument, "size_bytes must be > 0")
		}
		if f.GetSizeBytes() > maxSize {
			return nil, status.Error(codes.InvalidArgument, "file too large")
		}
		ext, ok := s3media.ContentTypeToExt(f.GetContentType())
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "unsupported content_type")
		}

		key := p.ObjectKey(environment, claims.ForwardedToken, ext, now)
		signed, err := p.PresignPut(ctx, key)
		if err != nil {
			return nil, status.Error(codes.Unavailable, "failed to sign upload url")
		}

		headers := map[string]string{}
		if ct := strings.TrimSpace(f.GetContentType()); ct != "" {
			headers["Content-Type"] = ct
		}
		out.Uploads = append(out.Uploads, &pb.UploadInit{
			S3Key:     signed.Key,
			UploadUrl: signed.URL,
			Headers:   headers,
		})
	}
	return out, nil
}

func (h *Handler) DeleteFoundItemImageUpload(ctx context.Context, req *pb.DeleteFoundItemImageUploadRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(req.GetS3Key())
	if key == "" {
		return nil, status.Error(codes.InvalidArgument, "s3_key is required")
	}
	if strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return nil, status.Error(codes.InvalidArgument, "invalid s3_key")
	}

	p, err := s3media.GetPresigner(ctx)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	environment := env.GetString("ENVIRONMENT", "development")
	allowedPrefix := p.AllowedSessionPrefix(environment, claims.ForwardedToken)
	if !strings.HasPrefix(key, allowedPrefix) {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}
	if err := p.DeleteObject(ctx, key); err != nil {
		return nil, status.Error(codes.Unavailable, "failed to delete image")
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) SearchFoundItemMatchesByEmbedding(ctx context.Context, req *pb.SearchFoundItemMatchesByEmbeddingRequest) (*pb.SearchFoundItemMatchesByEmbeddingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if len(req.GetQueryEmbedding()) == 0 {
		return &pb.SearchFoundItemMatchesByEmbeddingResponse{Matches: []*pb.FoundItemMatch{}}, nil
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 10
	}

	matches, err := h.usecase.SearchFoundItemMatchesByEmbedding(ctx, inbound.SearchFoundItemMatchesByEmbeddingInput{
		QueryEmbedding: req.GetQueryEmbedding(),
		Limit:          limit,
		MinSimilarity:  req.GetMinSimilarity(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	out := make([]*pb.FoundItemMatch, 0, len(matches))
	for _, m := range matches {
		it := mapper.FoundItemToPB(&m.Item)
		attachPresignedImageURLs(ctx, it)
		out = append(out, &pb.FoundItemMatch{
			Item:            it,
			SimilarityScore: m.SimilarityScore,
		})
	}
	return &pb.SearchFoundItemMatchesByEmbeddingResponse{Matches: out}, nil
}

func (h *Handler) ListClaims(ctx context.Context, req *pb.ListClaimsRequest) (*pb.ListClaimsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if _, err := requireStaffClaims(ctx); err != nil {
		return nil, err
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
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforceStaffIDMatch(req.GetStaffId(), claims); err != nil {
		return nil, err
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
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforceStaffIDMatch(req.GetStaffId(), claims); err != nil {
		return nil, err
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
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}
	if err := enforceStaffIDMatch(req.GetStaffId(), claims); err != nil {
		return nil, err
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
	claims, err := requireStaffClaims(ctx)
	if err != nil {
		return nil, err
	}

	createdBy := strings.TrimSpace(req.GetCreatedByStaffId())
	if createdBy != "" && createdBy != claims.StaffID {
		return nil, status.Error(codes.PermissionDenied, "created_by_staff_id mismatch")
	}
	createdBy = claims.StaffID

	routes, err := h.usecase.ListRoutes(ctx, inbound.ListRoutesInput{
		CreatedByStaffID: createdBy,
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
