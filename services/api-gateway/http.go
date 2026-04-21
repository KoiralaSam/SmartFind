package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	grpcclients "smartfind/services/api-gateway/grpc_clients"
	"smartfind/shared/env"
	"smartfind/shared/grpcclient"

	detailpb "smartfind/shared/proto/detailextractor"
	passengerpb "smartfind/shared/proto/passenger"
	staffpb "smartfind/shared/proto/staff"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// corsMiddleware adds CORS headers to allow requests from the web frontend
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from the Next.js frontend (localhost:5173)
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func forwardedTokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	// Prefer explicit Authorization header when present.
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return strings.TrimSpace(parts[1])
		}
		return authHeader
	}

	// Fall back to cookies used by the gateway login flows. When both staff and
	// passenger sessions exist in one browser, pick the cookie that matches the
	// request path so passenger APIs are not authenticated as staff.
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/passenger/"):
		if c, err := r.Cookie("passenger_session"); err == nil {
			if v := strings.TrimSpace(c.Value); v != "" {
				return v
			}
		}
	case strings.HasPrefix(path, "/staff/"):
		if c, err := r.Cookie("staff_session"); err == nil {
			if v := strings.TrimSpace(c.Value); v != "" {
				return v
			}
		}
	default:
		if c, err := r.Cookie("staff_session"); err == nil {
			if v := strings.TrimSpace(c.Value); v != "" {
				return v
			}
		}
		if c, err := r.Cookie("passenger_session"); err == nil {
			if v := strings.TrimSpace(c.Value); v != "" {
				return v
			}
		}
	}

	return ""
}

func extractDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DetailExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.ImageBase64) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "image_base64 is required"})
		return
	}

	internal := strings.TrimSpace(env.GetString("INTERNAL_SERVICE_SECRET", ""))
	if internal == "" {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "INTERNAL_SERVICE_SECRET is not configured"})
		return
	}
	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}

	client, err := grpcclients.NewDetailExtractorGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to detail extractor service"})
		return
	}
	defer client.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := client.Client.Extract(ctx, &detailpb.ExtractRequest{ImageBase64: req.ImageBase64})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DetailExtractResponse{
		ItemName:        resp.GetItemName(),
		ItemType:        resp.GetItemType(),
		Category:        resp.GetCategory(),
		Brand:           resp.GetBrand(),
		Model:           resp.GetModel(),
		Color:           resp.GetColor(),
		Material:        resp.GetMaterial(),
		ItemCondition:   resp.GetItemCondition(),
		ItemDescription: resp.GetItemDescription(),
	})
}

func passengerLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PassengerLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Fallback for form-encoded requests.
		req.IDToken = r.FormValue("id_token")
	}
	if req.IDToken == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "id_token is required"})
		return
	}

	passengerClient, err := grpcclients.NewPassengerGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to passenger service"})
		return
	}
	defer passengerClient.Close()

	resp, err := passengerClient.Client.Login(r.Context(), &passengerpb.LoginRequest{
		IdToken: req.IDToken,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	dto := PassengerLoginResponse{
		SessionToken: resp.GetSessionToken(),
	}
	if p := resp.GetPassenger(); p != nil {
		dto.Passenger = PassengerDTO{
			ID:        p.GetId(),
			Email:     p.GetEmail(),
			FullName:  p.GetFullName(),
			Phone:     p.GetPhone(),
			AvatarURL: p.GetAvatarUrl(),
		}
	}

	setPassengerSessionCookie(w, resp.GetSessionToken())
	writeJSON(w, http.StatusOK, dto)
}

func passengerLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clearPassengerSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func staffLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StaffLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Email = r.FormValue("email")
		req.Password = r.FormValue("password")
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "email and password are required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	resp, err := staffClient.Client.Login(r.Context(), &staffpb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	dto := StaffLoginResponse{
		SessionToken: resp.GetSessionToken(),
	}
	if s := resp.GetStaff(); s != nil {
		dto.Staff = StaffDTO{
			ID:        s.GetId(),
			FullName:  s.GetFullName(),
			Email:     s.GetEmail(),
			CreatedAt: timestampToTime(s.GetCreatedAt()),
			UpdatedAt: timestampToTime(s.GetUpdatedAt()),
		}
	}

	setStaffSessionCookie(w, resp.GetSessionToken())
	writeJSON(w, http.StatusOK, dto)
}

func staffLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clearStaffSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func staffCreateStaffHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StaffCreateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.TransitCode) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "transit_code is required"})
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "email is required"})
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "password is required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	resp, err := staffClient.Client.CreateStaff(r.Context(), &staffpb.CreateStaffRequest{
		TransitCode: req.TransitCode,
		FullName:    req.FullName,
		Email:       req.Email,
		Password:    req.Password,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	dto := StaffDTO{
		ID:        resp.GetId(),
		FullName:  resp.GetFullName(),
		Email:     resp.GetEmail(),
		CreatedAt: timestampToTime(resp.GetCreatedAt()),
		UpdatedAt: timestampToTime(resp.GetUpdatedAt()),
	}
	writeJSON(w, http.StatusOK, dto)
}

func staffCreateFoundItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StaffCreateFoundItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.StaffID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "staff_id is required"})
		return
	}
	if strings.TrimSpace(req.ItemName) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "item_name is required"})
		return
	}

	var ts *timestamppb.Timestamp
	if strings.TrimSpace(req.DateFound) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.DateFound))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "date_found must be RFC3339"})
			return
		}
		ts = timestamppb.New(t)
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.CreateFoundItem(ctx, &staffpb.CreateFoundItemRequest{
		StaffId:         req.StaffID,
		ItemName:        req.ItemName,
		ItemDescription: req.ItemDescription,
		ItemType:        req.ItemType,
		Brand:           req.Brand,
		Model:           req.Model,
		Color:           req.Color,
		Material:        req.Material,
		ItemCondition:   req.ItemCondition,
		Category:        req.Category,
		LocationFound:   req.LocationFound,
		RouteOrStation:  req.RouteOrStation,
		RouteId:         req.RouteID,
		DateFound:       ts,
		ImageKeys:       req.ImageKeys,
		PrimaryImageKey: req.PrimaryImageKey,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, foundItemPBToDTO(r.Context(), resp))
}

func staffUpdateFoundItemStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StaffUpdateFoundItemStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.StaffID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "staff_id is required"})
		return
	}
	if strings.TrimSpace(req.FoundItemID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "found_item_id is required"})
		return
	}
	if strings.TrimSpace(req.Status) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "status is required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.UpdateFoundItemStatus(ctx, &staffpb.UpdateFoundItemStatusRequest{
		StaffId:     req.StaffID,
		FoundItemId: req.FoundItemID,
		Status:      req.Status,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, foundItemPBToDTO(r.Context(), resp))
}

func staffUpdateFoundItemHandler(w http.ResponseWriter, r *http.Request) {
	// Path: PUT /staff/found-items/{id}
	foundItemID := strings.TrimPrefix(r.URL.Path, "/staff/found-items/")
	foundItemID = strings.TrimSpace(foundItemID)
	if foundItemID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "found_item_id is required in path"})
		return
	}

	var req StaffUpdateFoundItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.StaffID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "staff_id is required"})
		return
	}

	pbReq := &staffpb.UpdateFoundItemRequest{
		StaffId:         req.StaffID,
		FoundItemId:     foundItemID,
		ItemName:        req.ItemName,
		ItemDescription: req.ItemDescription,
		ItemType:        req.ItemType,
		Brand:           req.Brand,
		Model:           req.Model,
		Color:           req.Color,
		Material:        req.Material,
		ItemCondition:   req.ItemCondition,
		Category:        req.Category,
		LocationFound:   req.LocationFound,
		RouteOrStation:  req.RouteOrStation,
		RouteId:         req.RouteID,
		ImageKeys:       req.ImageKeys,
		PrimaryImageKey: req.PrimaryImageKey,
	}
	if strings.TrimSpace(req.DateFound) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.DateFound))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "date_found must be RFC3339"})
			return
		}
		pbReq.DateFound = timestamppb.New(t)
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.UpdateFoundItem(ctx, pbReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, foundItemPBToDTO(r.Context(), resp))
}

func staffDeleteFoundItemHandler(w http.ResponseWriter, r *http.Request) {
	// Path: DELETE /staff/found-items/{id}
	foundItemID := strings.TrimPrefix(r.URL.Path, "/staff/found-items/")
	foundItemID = strings.TrimSpace(foundItemID)
	if foundItemID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "found_item_id is required in path"})
		return
	}

	var req StaffDeleteFoundItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.StaffID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "staff_id is required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	_, err = staffClient.Client.DeleteFoundItem(ctx, &staffpb.DeleteFoundItemRequest{
		StaffId:     req.StaffID,
		FoundItemId: foundItemID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func staffListFoundItemsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := queryInt(r, "limit", 0)
	offset := queryInt(r, "offset", 0)

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.ListFoundItems(ctx, &staffpb.ListFoundItemsRequest{
		Status:          r.URL.Query().Get("status"),
		RouteId:         r.URL.Query().Get("route_id"),
		PostedByStaffId: r.URL.Query().Get("posted_by_staff_id"),
		Limit:           int32(limit),
		Offset:          int32(offset),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	items := make([]FoundItemDTO, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		items = append(items, foundItemPBToDTO(r.Context(), it))
	}
	writeJSON(w, http.StatusOK, StaffListFoundItemsResponse{Items: items})
}

func staffListClaimsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := queryInt(r, "limit", 0)
	offset := queryInt(r, "offset", 0)

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.ListClaims(ctx, &staffpb.ListClaimsRequest{
		Status:      r.URL.Query().Get("status"),
		ItemId:      r.URL.Query().Get("item_id"),
		PassengerId: r.URL.Query().Get("passenger_id"),
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	claims := make([]ItemClaimDTO, 0, len(resp.GetClaims()))
	passengerIDs := make([]string, 0, len(resp.GetClaims()))
	for _, c := range resp.GetClaims() {
		dto := itemClaimPBToDTO(c)
		claims = append(claims, dto)
		if strings.TrimSpace(dto.ClaimantPassengerID) != "" {
			passengerIDs = append(passengerIDs, dto.ClaimantPassengerID)
		}
	}
	passengerProfiles := loadPassengerProfilesByIDsViaRPC(r.Context(), forwarded, passengerIDs)
	for i := range claims {
		p := passengerProfiles[strings.TrimSpace(claims[i].ClaimantPassengerID)]
		if strings.TrimSpace(claims[i].ClaimantName) == "" {
			claims[i].ClaimantName = p.Name
		}
		if strings.TrimSpace(claims[i].ClaimantEmail) == "" {
			claims[i].ClaimantEmail = p.Email
		}
	}
	writeJSON(w, http.StatusOK, StaffListClaimsResponse{Claims: claims})
}

func staffReviewClaimHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StaffReviewClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.StaffID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "staff_id is required"})
		return
	}
	if strings.TrimSpace(req.ClaimID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "claim_id is required"})
		return
	}
	if strings.TrimSpace(req.Decision) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "decision is required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.ReviewClaim(ctx, &staffpb.ReviewClaimRequest{
		StaffId:  req.StaffID,
		ClaimId:  req.ClaimID,
		Decision: req.Decision,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, itemClaimPBToDTO(resp))
}

func staffCreateRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StaffCreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.StaffID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "staff_id is required"})
		return
	}
	if strings.TrimSpace(req.RouteName) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "route_name is required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.CreateRoute(ctx, &staffpb.CreateRouteRequest{
		StaffId:   req.StaffID,
		RouteName: req.RouteName,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, routePBToDTO(resp))
}

func staffDeleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StaffDeleteRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.StaffID = r.URL.Query().Get("staff_id")
		req.RouteID = r.URL.Query().Get("route_id")
	}
	if strings.TrimSpace(req.StaffID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "staff_id is required"})
		return
	}
	if strings.TrimSpace(req.RouteID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "route_id is required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	_, err = staffClient.Client.DeleteRoute(ctx, &staffpb.DeleteRouteRequest{
		StaffId: req.StaffID,
		RouteId: req.RouteID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func staffListRoutesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := queryInt(r, "limit", 0)
	offset := queryInt(r, "offset", 0)

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)

	resp, err := staffClient.Client.ListRoutes(ctx, &staffpb.ListRoutesRequest{
		CreatedByStaffId: r.URL.Query().Get("created_by_staff_id"),
		Limit:            int32(limit),
		Offset:           int32(offset),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	routes := make([]RouteDTO, 0, len(resp.GetRoutes()))
	for _, rt := range resp.GetRoutes() {
		routes = append(routes, routePBToDTO(rt))
	}
	writeJSON(w, http.StatusOK, StaffListRoutesResponse{Routes: routes})
}

func setPassengerSessionCookie(w http.ResponseWriter, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	maxAge := env.GetInt("JWT_TTL_SECONDS", 0)
	if maxAge <= 0 {
		maxAge = int((7 * 24 * time.Hour) / time.Second)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "passenger_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   env.GetBool("COOKIE_SECURE", false),
		MaxAge:   maxAge,
	})
}

func clearPassengerSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "passenger_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   env.GetBool("COOKIE_SECURE", false),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func setStaffSessionCookie(w http.ResponseWriter, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	maxAge := env.GetInt("JWT_TTL_SECONDS", 0)
	if maxAge <= 0 {
		maxAge = int((7 * 24 * time.Hour) / time.Second)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "staff_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   env.GetBool("COOKIE_SECURE", false),
		MaxAge:   maxAge,
	})
}

func clearStaffSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "staff_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   env.GetBool("COOKIE_SECURE", false),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func queryInt(r *http.Request, key string, def int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

func timestampToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func foundItemPBToDTO(ctx context.Context, it *staffpb.FoundItem) FoundItemDTO {
	if it == nil {
		return FoundItemDTO{}
	}
	dto := FoundItemDTO{
		ID:              it.GetId(),
		PostedByStaffID: it.GetPostedByStaffId(),
		ItemName:        it.GetItemName(),
		ItemDescription: it.GetItemDescription(),
		ItemType:        it.GetItemType(),
		Brand:           it.GetBrand(),
		Model:           it.GetModel(),
		Color:           it.GetColor(),
		Material:        it.GetMaterial(),
		ItemCondition:   it.GetItemCondition(),
		Category:        it.GetCategory(),
		LocationFound:   it.GetLocationFound(),
		RouteOrStation:  it.GetRouteOrStation(),
		RouteID:         it.GetRouteId(),
		DateFound:       timestampToTime(it.GetDateFound()),
		Status:          it.GetStatus(),
		CreatedAt:       timestampToTime(it.GetCreatedAt()),
		UpdatedAt:       timestampToTime(it.GetUpdatedAt()),
	}

	urls := it.GetImageUrls()
	if len(urls) > 0 {
		dto.Images = urls
	}

	primaryURL := strings.TrimSpace(it.GetPrimaryImageUrl())
	if primaryURL != "" {
		dto.Image = primaryURL
	} else if len(urls) > 0 {
		dto.Image = urls[0]
	}
	return dto
}

func itemClaimPBToDTO(c *staffpb.ItemClaim) ItemClaimDTO {
	if c == nil {
		return ItemClaimDTO{}
	}
	return ItemClaimDTO{
		ID:                  c.GetId(),
		ItemID:              c.GetItemId(),
		ClaimantPassengerID: c.GetClaimantPassengerId(),
		LostReportID:        c.GetLostReportId(),
		Message:             c.GetMessage(),
		Status:              c.GetStatus(),
		CreatedAt:           timestampToTime(c.GetCreatedAt()),
		UpdatedAt:           timestampToTime(c.GetUpdatedAt()),
	}
}

type passengerProfile struct {
	Name  string
	Email string
}

func loadPassengerProfilesByIDsViaRPC(ctx context.Context, forwarded string, ids []string) map[string]passengerProfile {
	out := map[string]passengerProfile{}
	unique := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return out
	}
	forwarded = strings.TrimSpace(forwarded)
	if forwarded == "" {
		return out
	}

	passengerClient, err := grpcclients.NewPassengerGRPCClient()
	if err != nil {
		return out
	}
	defer passengerClient.Close()

	rpcCtx := grpcclient.WithForwardedToken(ctx, forwarded)
	for _, id := range unique {
		resp, err := passengerClient.Client.GetPassenger(rpcCtx, &passengerpb.GetPassengerRequest{
			PassengerId: id,
		})
		if err != nil || resp == nil {
			continue
		}
		out[id] = passengerProfile{
			Name:  strings.TrimSpace(resp.GetFullName()),
			Email: strings.TrimSpace(resp.GetEmail()),
		}
	}
	return out
}

func routePBToDTO(rt *staffpb.Route) RouteDTO {
	if rt == nil {
		return RouteDTO{}
	}
	return RouteDTO{
		ID:               rt.GetId(),
		RouteName:        rt.GetRouteName(),
		CreatedByStaffID: rt.GetCreatedByStaffId(),
		CreatedAt:        timestampToTime(rt.GetCreatedAt()),
		UpdatedAt:        timestampToTime(rt.GetUpdatedAt()),
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: st.Message()})
	case codes.Unauthenticated:
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: st.Message()})
	case codes.PermissionDenied:
		writeJSON(w, http.StatusForbidden, ErrorResponse{Error: st.Message()})
	case codes.NotFound:
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: st.Message()})
	default:
		writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: st.Message()})
	}
}
