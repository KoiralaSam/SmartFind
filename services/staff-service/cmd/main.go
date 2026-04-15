package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smartfind/shared/db"
	"smartfind/shared/env"

	"google.golang.org/grpc"
)

func main() {
	grpcAddr := env.GetString("STAFF_GRPC_ADDR", ":50052")
	dbURL := env.GetString("DATABASE_URL", "")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if err := db.InitDB(ctx, dbURL); err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	defer func() {
		if db.Pool != nil {
			db.Pool.Close()
		}
	}()

	grpcServer := grpc.NewServer()

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	go func() {
		log.Printf("Staff gRPC listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("grpc serve error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down staff gRPC server...")

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		grpcServer.Stop()
	}
}
