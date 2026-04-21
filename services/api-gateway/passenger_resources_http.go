package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	grpcclients "smartfind/services/api-gateway/grpc_clients"
	"smartfind/shared/env"
	"smartfind/shared/grpcclient"

	passengerpb "smartfind/shared/proto/passenger"
)

type PassengerLostReportDTO struct {
	ID             string    `json:"id"`
	ItemName       string    `json:"item_name"`
	Status         string    `json:"status"`
	RouteOrStation string    `json:"route_or_station"`
	DateLost       time.Time `json:"date_lost"`
	CreatedAt      time.Time `json:"created_at"`
}

type PassengerListLostReportsResponse struct {
	Reports []PassengerLostReportDTO `json:"reports"`
}

type PassengerListClaimsResponse struct {
	Claims []ItemClaimDTO `json:"claims"`
}

func passengerListLostReportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}

	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

	client, err := grpcclients.NewPassengerGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to passenger service"})
		return
	}
	defer client.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := client.Client.ListLostReports(ctx, &passengerpb.ListLostReportsRequest{
		PassengerId: "",
		Status:      statusFilter,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	out := make([]PassengerLostReportDTO, 0, len(resp.GetReports()))
	for _, rpt := range resp.GetReports() {
		if rpt == nil {
			continue
		}
		var dateLost time.Time
		if rpt.GetDateLost() != nil {
			dateLost = rpt.GetDateLost().AsTime()
		}
		var createdAt time.Time
		if rpt.GetCreatedAt() != nil {
			createdAt = rpt.GetCreatedAt().AsTime()
		}
		out = append(out, PassengerLostReportDTO{
			ID:             rpt.GetId(),
			ItemName:       rpt.GetItemName(),
			Status:         rpt.GetStatus(),
			RouteOrStation: rpt.GetRouteOrStation(),
			DateLost:       dateLost,
			CreatedAt:      createdAt,
		})
	}

	writeJSON(w, http.StatusOK, PassengerListLostReportsResponse{Reports: out})
}

func passengerListMyClaimsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}

	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	client, err := grpcclients.NewPassengerGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to passenger service"})
		return
	}
	defer client.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := client.Client.ListMyClaims(ctx, &passengerpb.ListMyClaimsRequest{
		Status: statusFilter,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Enrich with found-item details + presigned image URLs (same shape as match cards)
	// without changing passenger.proto: reuse SearchFoundItemMatches per lost report.
	matchByFoundItemID := map[string]*passengerpb.FoundItemMatch{}
	notificationByFoundItemID := map[string]*passengerpb.PassengerMatchNotification{}
	seenLostReport := map[string]struct{}{}
	for _, c := range resp.GetClaims() {
		if c == nil {
			continue
		}
		lr := strings.TrimSpace(c.GetLostReportId())
		if lr == "" {
			continue
		}
		if _, ok := seenLostReport[lr]; ok {
			continue
		}
		seenLostReport[lr] = struct{}{}
		mr, err := client.Client.SearchFoundItemMatches(ctx, &passengerpb.SearchFoundItemMatchesRequest{
			LostReportId: lr,
			Limit:        100,
		})
		if err != nil {
			continue
		}
		for _, m := range mr.GetMatches() {
			if m == nil {
				continue
			}
			fid := strings.TrimSpace(m.GetFoundItemId())
			if fid == "" {
				continue
			}
			matchByFoundItemID[fid] = m
		}
	}
	// Fallback enrichment source: notifications keep item name/image snapshots
	// even after the found item is no longer returned by match search.
	if nr, err := client.Client.ListNotifications(ctx, &passengerpb.ListNotificationsRequest{Limit: 200}); err == nil {
		for _, n := range nr.GetNotifications() {
			if n == nil {
				continue
			}
			fid := strings.TrimSpace(n.GetFoundItemId())
			if fid == "" {
				continue
			}
			notificationByFoundItemID[fid] = n
		}
	}
	foundItemStatusByID := loadFoundItemStatusByIDs(r.Context(), resp.GetClaims())

	claims := make([]ItemClaimDTO, 0, len(resp.GetClaims()))
	for _, c := range resp.GetClaims() {
		if c == nil {
			continue
		}
		var createdAt time.Time
		if c.GetCreatedAt() != nil {
			createdAt = c.GetCreatedAt().AsTime()
		}
		var updatedAt time.Time
		if c.GetUpdatedAt() != nil {
			updatedAt = c.GetUpdatedAt().AsTime()
		}
		dto := ItemClaimDTO{
			ID:                  c.GetId(),
			ItemID:              c.GetItemId(),
			ClaimantPassengerID: c.GetClaimantPassengerId(),
			LostReportID:        c.GetLostReportId(),
			Message:             c.GetMessage(),
			Status:              c.GetStatus(),
			CreatedAt:           createdAt,
			UpdatedAt:           updatedAt,
		}
		if m := matchByFoundItemID[strings.TrimSpace(c.GetItemId())]; m != nil {
			dto.FoundItem = passengerFoundItemMatchPBToDTO(m)
		} else if n := notificationByFoundItemID[strings.TrimSpace(c.GetItemId())]; n != nil {
			dto.FoundItem = passengerFoundItemFromNotificationPBToDTO(n, c.GetItemId(), c.GetStatus())
		}
		if dto.FoundItem != nil {
			claimStatus := strings.ToLower(strings.TrimSpace(dto.Status))
			if claimStatus == "" || claimStatus == "pending" {
				dto.Status = "matched"
			}
		}
		if dto.FoundItem != nil && strings.EqualFold(strings.TrimSpace(dto.FoundItem.Status), "claimed") {
			// Passenger-facing UX: once staff marks the item claimed/returned,
			// reflect that terminal state directly on the claim card.
			dto.Status = "claimed"
		}
		if strings.EqualFold(strings.TrimSpace(foundItemStatusByID[strings.TrimSpace(dto.ItemID)]), "claimed") {
			dto.Status = "claimed"
			if dto.FoundItem != nil {
				dto.FoundItem.Status = "claimed"
			}
		}
		claims = append(claims, dto)
	}
	writeJSON(w, http.StatusOK, PassengerListClaimsResponse{Claims: claims})
}

func loadFoundItemStatusByIDs(ctx context.Context, claims []*passengerpb.ItemClaim) map[string]string {
	out := map[string]string{}
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(claims))
	for _, c := range claims {
		if c == nil {
			continue
		}
		id := strings.TrimSpace(c.GetItemId())
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return out
	}

	dbURL := strings.TrimSpace(env.GetString("DATABASE_URL", ""))
	if dbURL == "" {
		return out
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return out
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		SELECT id::text, status::text
		FROM found_items
		WHERE id::text = ANY($1::text[])
	`, ids)
	if err != nil {
		return out
	}
	defer rows.Close()

	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			continue
		}
		out[strings.TrimSpace(id)] = strings.TrimSpace(status)
	}
	return out
}

func passengerFoundItemMatchPBToDTO(m *passengerpb.FoundItemMatch) *PassengerClaimFoundItemDTO {
	if m == nil {
		return nil
	}
	var df time.Time
	if m.GetDateFound() != nil {
		df = m.GetDateFound().AsTime()
	}
	return &PassengerClaimFoundItemDTO{
		FoundItemID:     m.GetFoundItemId(),
		ItemName:        m.GetItemName(),
		ItemDescription: m.GetItemDescription(),
		ItemType:        m.GetItemType(),
		Brand:           m.GetBrand(),
		Model:           m.GetModel(),
		Color:           m.GetColor(),
		Material:        m.GetMaterial(),
		ItemCondition:   m.GetItemCondition(),
		Category:        m.GetCategory(),
		LocationFound:   m.GetLocationFound(),
		RouteOrStation:  m.GetRouteOrStation(),
		RouteID:         m.GetRouteId(),
		DateFound:       df,
		Status:          m.GetStatus(),
		SimilarityScore: m.GetSimilarityScore(),
		ImageURLs:       append([]string(nil), m.GetImageUrls()...),
		PrimaryImageURL: m.GetPrimaryImageUrl(),
	}
}

func passengerFoundItemFromNotificationPBToDTO(n *passengerpb.PassengerMatchNotification, itemID string, claimStatus string) *PassengerClaimFoundItemDTO {
	if n == nil {
		return nil
	}
	status := "unclaimed"
	switch strings.ToLower(strings.TrimSpace(claimStatus)) {
	case "approved":
		status = "claimed"
	case "rejected", "cancelled":
		status = "unclaimed"
	}
	urls := append([]string(nil), n.GetImageUrls()...)
	primary := strings.TrimSpace(n.GetPrimaryImageUrl())
	if primary == "" && len(urls) > 0 {
		primary = urls[0]
	}
	return &PassengerClaimFoundItemDTO{
		FoundItemID:     strings.TrimSpace(itemID),
		ItemName:        n.GetItemName(),
		Status:          status,
		SimilarityScore: n.GetSimilarityScore(),
		ImageURLs:       urls,
		PrimaryImageURL: primary,
	}
}

type PassengerFileClaimRequest struct {
	FoundItemID  string `json:"found_item_id"`
	LostReportID string `json:"lost_report_id"`
	Message      string `json:"message"`
}

// passengerFileClaimHandler lets a logged-in passenger file a claim from the
// notifications drawer or any other canonical UI surface. The gateway resolves
// the passenger from the forwarded session token so the client never sends a
// passenger_id it could have forged; the passenger-service FileClaim handler
// enforces the same invariant on the gRPC side.
func passengerFileClaimHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	forwarded := forwardedTokenFromRequest(r)
	if forwarded == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}

	var req PassengerFileClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if strings.TrimSpace(req.FoundItemID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "found_item_id is required"})
		return
	}
	if strings.TrimSpace(req.LostReportID) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "lost_report_id is required"})
		return
	}

	client, err := grpcclients.NewPassengerGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to passenger service"})
		return
	}
	defer client.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := client.Client.FileClaim(ctx, &passengerpb.FileClaimRequest{
		FoundItemId:  strings.TrimSpace(req.FoundItemID),
		LostReportId: strings.TrimSpace(req.LostReportID),
		Message:      strings.TrimSpace(req.Message),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	var createdAt time.Time
	if resp.GetCreatedAt() != nil {
		createdAt = resp.GetCreatedAt().AsTime()
	}
	var updatedAt time.Time
	if resp.GetUpdatedAt() != nil {
		updatedAt = resp.GetUpdatedAt().AsTime()
	}
	writeJSON(w, http.StatusOK, ItemClaimDTO{
		ID:                  resp.GetId(),
		ItemID:              resp.GetItemId(),
		ClaimantPassengerID: resp.GetClaimantPassengerId(),
		LostReportID:        resp.GetLostReportId(),
		Message:             resp.GetMessage(),
		Status:              resp.GetStatus(),
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	})
}
