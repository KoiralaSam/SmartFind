package grpcclients

import (
	"smartfind/shared/env"
	pb "smartfind/shared/proto/passenger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PassengerGRPCClient struct {
	conn   *grpc.ClientConn
	Client pb.PassengerServiceClient
}

func NewPassengerGRPCClient() (*PassengerGRPCClient, error) {
	addr := env.GetString("PASSENGER_SERVICE_ADDRESS", "localhost:50051")
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &PassengerGRPCClient{
		conn:   conn,
		Client: pb.NewPassengerServiceClient(conn),
	}, nil
}

func (c *PassengerGRPCClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	_ = c.conn.Close()
}
