package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	grpcclients "smartfind/services/api-gateway/grpc_clients"
	"smartfind/shared/grpcclient"
	staffpb "smartfind/shared/proto/staff"
)

type deleteUploadRequest struct {
	S3Key string `json:"s3_key"`
}

// mediaDeleteUploadHandler deletes an uploaded image via staff-service.
// staff-service owns the S3 logic and enforces key scoping.
func mediaDeleteUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := forwardedTokenFromRequest(r)
	if strings.TrimSpace(token) == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing session token"})
		return
	}

	var req deleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	key := strings.TrimSpace(req.S3Key)
	if key == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "s3_key is required"})
		return
	}
	// Basic path hardening.
	if strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid s3_key"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), token)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	_, err = staffClient.Client.DeleteFoundItemImageUpload(ctx, &staffpb.DeleteFoundItemImageUploadRequest{
		S3Key: key,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
