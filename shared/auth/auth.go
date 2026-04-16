package auth

import (
	"context"
	"errors"
	"strings"

	"smartfind/shared/jwt"
)

type ActorType string

const (
	ActorStaff ActorType = "staff"
)

type Claims struct {
	ActorType      ActorType
	StaffID        string
	Email          string
	ForwardedToken string
}

var ErrNoClaims = errors.New("no verified claims")

type ctxKey int

const claimsKey ctxKey = iota

func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFromContext(ctx context.Context) (Claims, error) {
	if ctx == nil {
		return Claims{}, ErrNoClaims
	}
	v := ctx.Value(claimsKey)
	c, ok := v.(Claims)
	if !ok {
		return Claims{}, ErrNoClaims
	}
	if strings.TrimSpace(c.StaffID) == "" {
		return Claims{}, ErrNoClaims
	}
	return c, nil
}

func VerifyStaffSessionTokenFromEnv(token string) (Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Claims{}, errors.New("missing token")
	}
	c, err := jwt.VerifyTokenFromEnv(token)
	if err != nil {
		return Claims{}, err
	}
	return Claims{
		ActorType:      ActorStaff,
		StaffID:        c.Sub,
		Email:          c.Email,
		ForwardedToken: token,
	}, nil
}

