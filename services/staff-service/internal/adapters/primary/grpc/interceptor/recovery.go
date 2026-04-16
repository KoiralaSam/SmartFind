package interceptor

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Recovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in grpc method=%s: %v", info.FullMethod, r)
				err = status.Error(codes.Internal, "internal error")
				resp = nil
			}
		}()
		return handler(ctx, req)
	}
}
