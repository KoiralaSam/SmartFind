package grpcclient

import (
	"context"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxKey int

const forwardedTokenKey ctxKey = iota

func WithForwardedToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, forwardedTokenKey, strings.TrimSpace(token))
}

func ForwardedTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v := ctx.Value(forwardedTokenKey)
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

// UnaryAuthInterceptor attaches x-internal-token and x-forwarded-token metadata
// for internal service-to-service calls.
//
// - x-internal-token comes from INTERNAL_SERVICE_SECRET env var.
// - x-forwarded-token comes from the context via WithForwardedToken.
func UnaryAuthInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		internal := strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_SECRET"))
		forwarded := ForwardedTokenFromContext(ctx)

		// Attach only when values are present; servers can enforce per-method.
		pairs := make([]string, 0, 4)
		if internal != "" {
			pairs = append(pairs, "x-internal-token", internal)
		}
		if forwarded != "" {
			pairs = append(pairs, "x-forwarded-token", forwarded)
		}
		if len(pairs) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, pairs...)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
