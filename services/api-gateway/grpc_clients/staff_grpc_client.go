package grpcclients

import (
	"smartfind/shared/env"
	"smartfind/shared/grpcclient"
	pb "smartfind/shared/proto/staff"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StaffGRPCClient struct {
	conn   *grpc.ClientConn
	Client pb.StaffServiceClient
}

func NewStaffGRPCClient() (*StaffGRPCClient, error) {
	addr := env.GetString("STAFF_SERVICE_ADDRESS", "localhost:50052")
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcclient.UnaryAuthInterceptor()),
	)
	if err != nil {
		return nil, err
	}
	return &StaffGRPCClient{
		conn:   conn,
		Client: pb.NewStaffServiceClient(conn),
	}, nil
}

func (c *StaffGRPCClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	_ = c.conn.Close()
}
