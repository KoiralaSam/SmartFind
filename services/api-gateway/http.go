package main

import (
	"encoding/json"
	"net/http"

	grpcclients "smartfind/services/api-gateway/grpc_clients"

	pb "smartfind/shared/proto/passenger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	passengerClient, err := grpcclients.NewPassengerClient()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to connect to passenger service"})
		return
	}
	defer passengerClient.Close()

	resp, err := passengerClient.PassengerClient.Login(r.Context(), &pb.LoginRequest{
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

	writeJSON(w, http.StatusOK, dto)
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
