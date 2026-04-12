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

	// API routes with CORS (Go 1.22+ requires space between method and path)
	mux.HandleFunc("POST /passenger/login", corsMiddleware(passengerLoginHandler))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	log.Printf("API Gateway listening on %s", httpAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Http server error: %v", err)
	}
}
