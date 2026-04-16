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

	"google.golang.org/protobuf/types/known/timestamppb"
)

type PassengerNotificationDTO struct {
	ID              string    `json:"id"`
	LostReportID    string    `json:"lost_report_id"`
	FoundItemID     string    `json:"found_item_id"`
	SimilarityScore float64   `json:"similarity_score"`
	ItemName        string    `json:"item_name"`
	ImageURLs       []string  `json:"image_urls"`
	PrimaryImageURL string    `json:"primary_image_url"`
	CreatedAt       time.Time `json:"created_at"`
	ReadAt          time.Time `json:"read_at"`
}

type PassengerListNotificationsResponse struct {
	Notifications []PassengerNotificationDTO `json:"notifications"`
}

type PassengerMarkNotificationsReadRequest struct {
	NotificationIDs []string `json:"notification_ids"`
}

func passengerListNotificationsHandler(w http.ResponseWriter, r *http.Request) {
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

	limit := queryInt(r, "limit", 20)
	unreadOnly := strings.TrimSpace(r.URL.Query().Get("unread_only"))
	createdBeforeStr := strings.TrimSpace(r.URL.Query().Get("created_before"))

	var createdBefore *timestamppb.Timestamp
	if createdBeforeStr != "" {
		t, err := time.Parse(time.RFC3339, createdBeforeStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "created_before must be RFC3339"})
			return
		}
		createdBefore = timestamppb.New(t)
	}

	passengerClient, err := grpcclients.NewPassengerGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to passenger service"})
		return
	}
	defer passengerClient.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := passengerClient.Client.ListNotifications(ctx, &passengerpb.ListNotificationsRequest{
		Limit:         int32(limit),
		UnreadOnly:    strings.EqualFold(unreadOnly, "1") || strings.EqualFold(unreadOnly, "true"),
		CreatedBefore: createdBefore,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	out := make([]PassengerNotificationDTO, 0, len(resp.GetNotifications()))
	for _, n := range resp.GetNotifications() {
		if n == nil {
			continue
		}
		var createdAt time.Time
		if n.GetCreatedAt() != nil {
			createdAt = n.GetCreatedAt().AsTime()
		}
		var readAt time.Time
		if n.GetReadAt() != nil {
			readAt = n.GetReadAt().AsTime()
		}
		out = append(out, PassengerNotificationDTO{
			ID:              n.GetId(),
			LostReportID:    n.GetLostReportId(),
			FoundItemID:     n.GetFoundItemId(),
			SimilarityScore: n.GetSimilarityScore(),
			ItemName:        n.GetItemName(),
			ImageURLs:       n.GetImageUrls(),
			PrimaryImageURL: n.GetPrimaryImageUrl(),
			CreatedAt:       createdAt,
			ReadAt:          readAt,
		})
	}

	writeJSON(w, http.StatusOK, PassengerListNotificationsResponse{Notifications: out})
}

func passengerMarkNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
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

	var req PassengerMarkNotificationsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}

	passengerClient, err := grpcclients.NewPassengerGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to passenger service"})
		return
	}
	defer passengerClient.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), forwarded)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = passengerClient.Client.MarkNotificationRead(ctx, &passengerpb.MarkNotificationReadRequest{
		NotificationIds: req.NotificationIDs,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
