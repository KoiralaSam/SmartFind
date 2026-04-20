package main

import (
	"log"
	"net/http"

	"smartfind/shared/env"
)

var (
	httpAddr = env.GetString("GATEWAY_HTTP_ADDR", env.GetString("HTTP_ADDR", ":8081"))
)

func main() {
	log.Println("Starting API Gateway")

	mux := http.NewServeMux()

	// CORS preflight: OPTIONS must be handled for each path (browser sends OPTIONS before POST)
	optCORS := corsMiddleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("OPTIONS /api/v1/passenger/login", optCORS)
	mux.HandleFunc("OPTIONS /passenger/login", optCORS)
	mux.HandleFunc("OPTIONS /passenger/logout", optCORS)
	mux.HandleFunc("OPTIONS /passenger/notifications", optCORS)
	mux.HandleFunc("OPTIONS /passenger/notifications/read", optCORS)
	mux.HandleFunc("OPTIONS /passenger/lost-reports", optCORS)
	mux.HandleFunc("OPTIONS /passenger/claims", optCORS)
	mux.HandleFunc("OPTIONS /staff/login", optCORS)
	mux.HandleFunc("OPTIONS /staff/logout", optCORS)
	mux.HandleFunc("OPTIONS /staff", optCORS)
	mux.HandleFunc("OPTIONS /staff/found-items", optCORS)
	mux.HandleFunc("OPTIONS /staff/found-items/status", optCORS)
	mux.HandleFunc("OPTIONS /staff/claims", optCORS)
	mux.HandleFunc("OPTIONS /staff/claims/review", optCORS)
	mux.HandleFunc("OPTIONS /staff/routes", optCORS)
	mux.HandleFunc("OPTIONS /extract", optCORS)
	mux.HandleFunc("OPTIONS /media/uploads/init", optCORS)
	mux.HandleFunc("OPTIONS /media/uploads/delete", optCORS)

	// API routes with CORS (Go 1.22+ requires space between method and path)
	mux.HandleFunc("POST /passenger/login", corsMiddleware(passengerLoginHandler))
	mux.HandleFunc("POST /passenger/logout", corsMiddleware(passengerLogoutHandler))
	mux.HandleFunc("GET /passenger/notifications", corsMiddleware(passengerListNotificationsHandler))
	mux.HandleFunc("POST /passenger/notifications/read", corsMiddleware(passengerMarkNotificationsReadHandler))
	mux.HandleFunc("GET /passenger/lost-reports", corsMiddleware(passengerListLostReportsHandler))
	mux.HandleFunc("GET /passenger/claims", corsMiddleware(passengerListMyClaimsHandler))
	mux.HandleFunc("POST /passenger/claims", corsMiddleware(passengerFileClaimHandler))
	mux.HandleFunc("POST /staff/login", corsMiddleware(staffLoginHandler))
	mux.HandleFunc("POST /staff/logout", corsMiddleware(staffLogoutHandler))
	mux.HandleFunc("POST /staff", corsMiddleware(staffCreateStaffHandler))
	mux.HandleFunc("POST /staff/found-items", corsMiddleware(staffCreateFoundItemHandler))
	mux.HandleFunc("PUT /staff/found-items/status", corsMiddleware(staffUpdateFoundItemStatusHandler))
	mux.HandleFunc("GET /staff/found-items", corsMiddleware(staffListFoundItemsHandler))
	mux.HandleFunc("GET /staff/claims", corsMiddleware(staffListClaimsHandler))
	mux.HandleFunc("POST /staff/claims/review", corsMiddleware(staffReviewClaimHandler))
	mux.HandleFunc("POST /staff/routes", corsMiddleware(staffCreateRouteHandler))
	mux.HandleFunc("DELETE /staff/routes", corsMiddleware(staffDeleteRouteHandler))
	mux.HandleFunc("GET /staff/routes", corsMiddleware(staffListRoutesHandler))
	mux.HandleFunc("POST /extract", corsMiddleware(extractDetailsHandler))
	mux.HandleFunc("POST /media/uploads/init", corsMiddleware(mediaInitUploadsHandler))
	mux.HandleFunc("POST /media/uploads/delete", corsMiddleware(mediaDeleteUploadHandler))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	log.Printf("API Gateway listening on %s", httpAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Http server error: %v", err)
	}
}
