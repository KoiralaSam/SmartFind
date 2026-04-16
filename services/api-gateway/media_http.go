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

type initUploadFile struct {
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

type initUploadRequest struct {
	Files []initUploadFile `json:"files"`
}

type initUploadItem struct {
	S3Key     string            `json:"s3_key"`
	UploadURL string            `json:"upload_url"`
	Headers   map[string]string `json:"headers"`
}

type initUploadResponse struct {
	Uploads []initUploadItem `json:"uploads"`
}

func mediaInitUploadsHandler(w http.ResponseWriter, r *http.Request) {
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

	var req initUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	if len(req.Files) == 0 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "files is required"})
		return
	}

	staffClient, err := grpcclients.NewStaffGRPCClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to staff service"})
		return
	}
	defer staffClient.Close()

	files := make([]*staffpb.UploadFile, 0, len(req.Files))
	for _, f := range req.Files {
		files = append(files, &staffpb.UploadFile{
			ContentType: f.ContentType,
			SizeBytes:   f.SizeBytes,
		})
	}

	ctx := grpcclient.WithForwardedToken(r.Context(), token)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	resp, err := staffClient.Client.InitFoundItemImageUploads(ctx, &staffpb.InitFoundItemImageUploadsRequest{
		Files: files,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	out := initUploadResponse{Uploads: make([]initUploadItem, 0, len(resp.GetUploads()))}
	for _, u := range resp.GetUploads() {
		if u == nil {
			continue
		}
		out.Uploads = append(out.Uploads, initUploadItem{
			S3Key:     u.GetS3Key(),
			UploadURL: u.GetUploadUrl(),
			Headers:   u.GetHeaders(),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
