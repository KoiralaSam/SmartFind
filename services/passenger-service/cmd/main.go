package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	passengergrpc "smartfind/services/passenger-service/internal/adapters/primary/grpc"
	"smartfind/services/passenger-service/internal/adapters/secondary/postgres"
	"smartfind/services/passenger-service/internal/service"
	"smartfind/shared/db"
	"smartfind/shared/env"
)

func main() {
	grpcAddr := env.GetString("PASSENGER_GRPC_ADDR", ":50051")
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

	repo := postgres.NewPassengerRepository(db.GetDB())
	usecase := service.NewPassengerService(repo)

	grpcServer := passengergrpc.NewServer(usecase)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	go func() {
		log.Printf("Passenger gRPC listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("grpc serve error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down passenger gRPC server...")

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

