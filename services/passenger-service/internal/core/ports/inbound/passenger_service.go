package inbound

import (
	"context"
	"time"

	"smartfind/services/passenger-service/internal/core/domain"
)

type CreateLostReportInput struct {
	PassengerID     string
	ItemName        string
	ItemDescription string
	ItemType        string
	Brand           string
	Model           string
	Color           string
	Material        string
	ItemCondition   string
	Category        string
	LocationLost    string
	RouteOrStation  string
	RouteID         string
	DateLost        time.Time
}

type LostReport struct {
	ID                  string
	ReporterPassengerID string
	ItemName            string
	ItemDescription     string
	ItemType            string
	Brand               string
	Model               string
	Color               string
	Material            string
	ItemCondition       string
	Category            string
	LocationLost        string
	RouteOrStation      string
	RouteID             string
	DateLost            time.Time
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ListLostReportsInput struct {
	PassengerID string
	Status      string
}

type FoundItemMatch struct {
	FoundItemID     string
	ItemName        string
	ItemDescription string
	ItemType        string
	Brand           string
	Model           string
	Color           string
	Material        string
	ItemCondition   string
	Category        string
	LocationFound   string
	RouteOrStation  string
	RouteID         string
	DateFound       time.Time
	Status          string
	SimilarityScore float64
}

type SearchFoundItemsInput struct {
	PassengerID  string
	LostReportID string
	Limit        int
}

type FileClaimInput struct {
	PassengerID  string
	FoundItemID  string
	LostReportID string
	Message      string
}

type ItemClaim struct {
	ID                  string
	ItemID              string
	ClaimantPassengerID string
	LostReportID        string
	Message             string
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// PassengerUsecase defines the inbound application port for passenger operations.
type PassengerUsecase interface {
	Login(ctx context.Context, email string) (*domain.Passenger, error)
	CreateLostReport(ctx context.Context, in CreateLostReportInput) (*LostReport, error)
	ListLostReports(ctx context.Context, in ListLostReportsInput) ([]LostReport, error)
	DeleteLostReport(ctx context.Context, passengerID, lostReportID string) error
	SearchFoundItemMatches(ctx context.Context, in SearchFoundItemsInput) ([]FoundItemMatch, error)
	FileClaim(ctx context.Context, in FileClaimInput) (*ItemClaim, error)
}
