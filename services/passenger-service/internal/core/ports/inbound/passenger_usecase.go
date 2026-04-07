package inbound

import (
	"context"

	"smartfind/services/passenger-service/internal/core/domain"
)

type RegisterInput struct {
	Email    string
	Username string
	Password string
}

// PassengerUsecase defines the inbound application port for passenger operations.
type PassengerUsecase interface {
	Register(ctx context.Context, in RegisterInput) (*domain.Passenger, error)
	Login(ctx context.Context, email string) (*domain.Passenger, error)
}
