package grpcclients

import (
	"smartfind/shared/env"
	"smartfind/shared/grpcclient"
	pb "smartfind/shared/proto/detailextractor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DetailExtractorGRPCClient struct {
	conn   *grpc.ClientConn
	Client pb.DetailExtractorServiceClient
}

func NewDetailExtractorGRPCClient() (*DetailExtractorGRPCClient, error) {
	addr := env.GetString("DETAIL_EXTRACTER_SERVICE_ADDRESS", "localhost:50053")
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpcclient.UnaryAuthInterceptor()),
	)
	if err != nil {
		return nil, err
	}
	return &DetailExtractorGRPCClient{
		conn:   conn,
		Client: pb.NewDetailExtractorServiceClient(conn),
	}, nil
}

func (c *DetailExtractorGRPCClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	_ = c.conn.Close()
}
