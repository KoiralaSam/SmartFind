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

// UpdateLostReportInput patches a subset of fields on an existing lost
// report. Only non-nil pointers are applied, which lets callers
// distinguish "clear this field" (empty-string pointer) from "leave this
// field alone" (nil pointer). If any of the twelve embedded slots change,
// the passenger service recomputes the embedding.
type UpdateLostReportInput struct {
	PassengerID     string
	LostReportID    string
	ItemName        *string
	ItemDescription *string
	ItemType        *string
	Brand           *string
	Model           *string
	Color           *string
	Material        *string
	ItemCondition   *string
	Category        *string
	LocationLost    *string
	RouteOrStation  *string
	RouteID         *string
	DateLost        *time.Time
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
	ImageURLs       []string
	PrimaryImageURL string
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

type PassengerMatchNotification struct {
	ID              string
	PassengerID     string
	LostReportID    string
	FoundItemID     string
	SimilarityScore float64
	ItemName        string
	ImageURLs       []string
	PrimaryImageURL string
	CreatedAt       time.Time
	ReadAt          time.Time
}

type ListNotificationsInput struct {
	PassengerID   string
	Limit         int
	UnreadOnly    bool
	CreatedBefore time.Time
}

type MarkNotificationReadInput struct {
	PassengerID     string
	NotificationIDs []string
}

type ListMyClaimsInput struct {
	PassengerID string
	Status      string
	Limit       int
	Offset      int
}

// LoginInput is the Google Sign-In credential. The service verifies the ID token and upserts the passenger.
type LoginInput struct {
	IDToken string
}

// LoginResult is returned after a successful Google login; SessionToken is the app JWT for cookies.
type LoginResult struct {
	Passenger    *domain.Passenger
	SessionToken string
}

// PassengerUsecase defines the inbound application port for passenger operations.
type PassengerUsecase interface {
	// Login verifies the Google ID token (GOOGLE_CLIENT_ID), creates or updates the passenger row, and returns a JWT.
	Login(ctx context.Context, in LoginInput) (*LoginResult, error)
	GetPassengerByID(ctx context.Context, passengerID string) (*domain.Passenger, error)
	CreateLostReport(ctx context.Context, in CreateLostReportInput) (*LostReport, error)
	UpdateLostReport(ctx context.Context, in UpdateLostReportInput) (*LostReport, error)
	ListLostReports(ctx context.Context, in ListLostReportsInput) ([]LostReport, error)
	DeleteLostReport(ctx context.Context, passengerID, lostReportID string) error
	SearchFoundItemMatches(ctx context.Context, in SearchFoundItemsInput) ([]FoundItemMatch, error)
	FileClaim(ctx context.Context, in FileClaimInput) (*ItemClaim, error)
	ListNotifications(ctx context.Context, in ListNotificationsInput) ([]PassengerMatchNotification, error)
	MarkNotificationRead(ctx context.Context, in MarkNotificationReadInput) error
	ListMyClaims(ctx context.Context, in ListMyClaimsInput) ([]ItemClaim, error)
}
