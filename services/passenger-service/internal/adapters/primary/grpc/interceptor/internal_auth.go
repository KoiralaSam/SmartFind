package interceptor

import (
	"context"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func InternalAuth() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		expected := strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_SECRET"))
		if expected == "" {
			return nil, status.Error(codes.FailedPrecondition, "INTERNAL_SERVICE_SECRET not configured")
		}

		md, _ := metadata.FromIncomingContext(ctx)
		internal := strings.TrimSpace(firstMD(md, "x-internal-token"))
		if internal == "" || internal != expected {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid internal token")
		}
		return handler(ctx, req)
	}
}

func firstMD(md metadata.MD, key string) string {
	if md == nil {
		return ""
	}
	values := md.Get(key)
	if len(values) == 0 {
		values = md.Get(strings.ToLower(key))
	}
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
