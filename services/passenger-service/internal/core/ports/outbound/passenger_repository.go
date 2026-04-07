package outbound

import (
	"context"

	"smartfind/services/passenger-service/internal/core/domain"
)

// PassengerRepository defines the outbound persistence port for passengers.
type PassengerRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Passenger, error)
	GetByEmail(ctx context.Context, email string) (*domain.Passenger, error)
	Create(ctx context.Context, passenger domain.Passenger) error
	Update(ctx context.Context, passenger domain.Passenger) error
}
