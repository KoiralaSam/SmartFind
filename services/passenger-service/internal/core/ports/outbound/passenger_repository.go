package outbound

import (
	"context"
	"time"

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
	GetLostReportForPassenger(ctx context.Context, passengerID string, lostReportID string) (*inbound.LostReport, error)
	UpsertLostReportEmbedding(ctx context.Context, lostReportID string, embedding []float32) error
	GetLostReportEmbeddingForPassenger(ctx context.Context, passengerID string, lostReportID string) ([]float32, error)
	ListLostReports(ctx context.Context, passengerID string, status string) ([]inbound.LostReport, error)
	DeleteLostReport(ctx context.Context, passengerID string, lostReportID string) error
	UpdateLostReportStatus(ctx context.Context, passengerID string, lostReportID string, status string) error

	CreateItemClaim(ctx context.Context, claim inbound.ItemClaim) (*inbound.ItemClaim, error)
	CreateItemClaimAndMarkLostReportMatched(ctx context.Context, claim inbound.ItemClaim) (*inbound.ItemClaim, error)
	ListMyClaims(ctx context.Context, passengerID string, status string, limit int, offset int) ([]inbound.ItemClaim, error)

	ListNotifications(ctx context.Context, passengerID string, limit int, unreadOnly bool, createdBefore time.Time) ([]inbound.PassengerMatchNotification, error)
	MarkNotificationsRead(ctx context.Context, passengerID string, notificationIDs []string) error
}
