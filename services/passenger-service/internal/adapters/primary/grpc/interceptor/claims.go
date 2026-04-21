package interceptor

import (
	"context"
	"strings"

	"smartfind/shared/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func Claims(publicMethods map[string]bool) grpc.UnaryServerInterceptor {
	if publicMethods == nil {
		publicMethods = map[string]bool{}
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info != nil && publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, _ := metadata.FromIncomingContext(ctx)
		forwarded := strings.TrimSpace(firstMD(md, "x-forwarded-token"))
		if forwarded == "" {
			return nil, status.Error(codes.Unauthenticated, "missing forwarded token")
		}

		claims, err := auth.VerifyPassengerSessionTokenFromEnv(forwarded)
		if err != nil {
			claims, err = auth.VerifyStaffSessionTokenFromEnv(forwarded)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "invalid forwarded token")
			}
		}

		ctx = auth.WithClaims(ctx, claims)
		return handler(ctx, req)
	}
}
