package grpcadapter

import (
	"smartfind/shared/env"
	"smartfind/shared/grpcclient"
	staffpb "smartfind/shared/proto/staff"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StaffClient struct {
	conn   *grpc.ClientConn
	Client staffpb.StaffServiceClient
}

func NewStaffClient() (*StaffClient, error) {
	addr := env.GetString("STAFF_SERVICE_ADDRESS", "staff-service:50052")
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcclient.UnaryAuthInterceptor()),
	)
	if err != nil {
		return nil, err
	}
	return &StaffClient{
		conn:   conn,
		Client: staffpb.NewStaffServiceClient(conn),
	}, nil
}

func (c *StaffClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	_ = c.conn.Close()
}
