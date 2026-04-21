package outbound

import (
	"context"
	"errors"
	"time"

	"smartfind/services/passenger-service/internal/core/domain"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
)

// ErrLostReportHasActiveClaims is returned when a lost report cannot be deleted
// because it still has claims that are pending or approved.
var ErrLostReportHasActiveClaims = errors.New("lost report has active claims and cannot be deleted")

// ErrLostReportNotFound is returned when the target report doesn't exist for the passenger.
var ErrLostReportNotFound = errors.New("lost report not found")

// PassengerRepository defines the outbound persistence port for passengers.
type PassengerRepository interface {
	GetByID(ctx context.Context, id string) (*domain.Passenger, error)
	GetByEmail(ctx context.Context, email string) (*domain.Passenger, error)
	Create(ctx context.Context, passenger domain.Passenger) (*domain.Passenger, error)
	Update(ctx context.Context, passenger domain.Passenger) error

	CreateLostReport(ctx context.Context, report inbound.LostReport) (*inbound.LostReport, error)
	UpdateLostReport(ctx context.Context, in inbound.UpdateLostReportInput) (*inbound.LostReport, error)
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
