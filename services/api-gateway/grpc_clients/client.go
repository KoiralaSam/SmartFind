package grpcclients

import (
	"smartfind/shared/env"
	pb "smartfind/shared/proto/passenger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type passengerClient struct {
	conn            *grpc.ClientConn
	PassengerClient pb.PassengerServiceClient
}

func NewPassengerClient() (*passengerClient, error) {
	passengerServiceAddress := env.GetString("PASSENGER_SERVICE_ADDRESS", "localhost:50051")
	conn, err := grpc.NewClient(passengerServiceAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &passengerClient{
		conn:            conn,
		PassengerClient: pb.NewPassengerServiceClient(conn),
	}, nil
}

func (c *passengerClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}
