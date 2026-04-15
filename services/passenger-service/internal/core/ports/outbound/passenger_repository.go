package outbound

import (
	"context"

	"smartfind/services/passenger-service/internal/core/domain"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
)

// PassengerRepository defines the outbound persistence port for passengers.
type PassengerRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Passenger, error)
	GetByEmail(ctx context.Context, email string) (*domain.Passenger, error)
	Create(ctx context.Context, passenger domain.Passenger) (*domain.Passenger, error)
	Update(ctx context.Context, passenger domain.Passenger) error

	CreateLostReport(ctx context.Context, report inbound.LostReport) (*inbound.LostReport, error)
	UpsertLostReportEmbedding(ctx context.Context, lostReportID string, embedding []float32) error
	ListLostReports(ctx context.Context, passengerID string, status string) ([]inbound.LostReport, error)
	DeleteLostReport(ctx context.Context, passengerID string, lostReportID string) error

	SearchFoundItemMatches(ctx context.Context, passengerID string, lostReportID string, limit int) ([]inbound.FoundItemMatch, error)
	CreateItemClaim(ctx context.Context, claim inbound.ItemClaim) (*inbound.ItemClaim, error)
}
