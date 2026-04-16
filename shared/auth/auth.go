package auth

import (
	"context"
	"errors"
	"strings"

	"smartfind/shared/jwt"
)

type ActorType string

const (
	ActorPassenger ActorType = "passenger"
	ActorStaff     ActorType = "staff"
)

type Claims struct {
	ActorType      ActorType
	PassengerID    string
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
	if strings.TrimSpace(string(c.ActorType)) == "" {
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
	if strings.TrimSpace(c.ActorType) != string(ActorStaff) {
		return Claims{}, errors.New("invalid actorType")
	}
	return Claims{
		ActorType:      ActorStaff,
		StaffID:        c.Sub,
		Email:          c.Email,
		ForwardedToken: token,
	}, nil
}

func VerifyPassengerSessionTokenFromEnv(token string) (Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Claims{}, errors.New("missing token")
	}
	c, err := jwt.VerifyTokenFromEnv(token)
	if err != nil {
		return Claims{}, err
	}
	if strings.TrimSpace(c.ActorType) != string(ActorPassenger) {
		return Claims{}, errors.New("invalid actorType")
	}
	return Claims{
		ActorType:      ActorPassenger,
		PassengerID:    c.Sub,
		Email:          c.Email,
		ForwardedToken: token,
	}, nil
}
