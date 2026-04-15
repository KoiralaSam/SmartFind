package interceptor

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func Logging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Printf("grpc method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
		return resp, err
	}
}

