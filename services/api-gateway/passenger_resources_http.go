package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	grpcclients "smartfind/services/api-gateway/grpc_clients"
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
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
		claims = append(claims, ItemClaimDTO{
			ID:                  c.GetId(),
			ItemID:              c.GetItemId(),
			ClaimantPassengerID: c.GetClaimantPassengerId(),
			LostReportID:        c.GetLostReportId(),
			Message:             c.GetMessage(),
			Status:              c.GetStatus(),
			CreatedAt:           createdAt,
			UpdatedAt:           updatedAt,
		})
	}
	writeJSON(w, http.StatusOK, PassengerListClaimsResponse{Claims: claims})
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
