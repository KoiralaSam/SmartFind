package inbound

import (
	"context"
	"time"
)

// Staff represents a staff member (table: staff).
// This is intentionally minimal; callers should treat it as a data carrier.
type Staff struct {
	ID        string
	FullName  string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	Staff        *Staff
	SessionToken string
}

// CreateStaffInput creates a new staff user (table: staff).
// Password is optional here; it can be set via SetPassword.
type CreateStaffInput struct {
	TransitCode string
	FullName    string
	Email       string
	Password    string
}

// FoundItem mirrors the found_items table as evolved by migrations 000005 and 000006.
type FoundItem struct {
	ID              string
	PostedByStaffID string
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
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ImageKeys       []string
	PrimaryImageKey string
}

type CreateFoundItemInput struct {
	StaffID         string
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
	ImageKeys       []string
	PrimaryImageKey string
}

type UpdateFoundItemStatusInput struct {
	StaffID     string
	FoundItemID string
	Status      string // found_item_status: unclaimed|claimed|returned|archived
}

type ListFoundItemsInput struct {
	Status          string
	RouteID         string
	PostedByStaffID string
	Limit           int
	Offset          int
}

type SearchFoundItemMatchesByEmbeddingInput struct {
	QueryEmbedding []float32
	Limit          int
	MinSimilarity  float64
}

type FoundItemMatch struct {
	Item            FoundItem
	SimilarityScore float64
}

// ItemClaim mirrors item_claims table (migrations 000001).
type ItemClaim struct {
	ID                  string
	ItemID              string
	ClaimantPassengerID string
	LostReportID        string
	Message             string
	Status              string // claim_status: pending|approved|rejected|cancelled
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ListClaimsInput struct {
	Status      string
	ItemID      string
	PassengerID string
	Limit       int
	Offset      int
}

type ReviewClaimInput struct {
	StaffID string
	ClaimID string
	// Decision must be "approved" or "rejected".
	Decision string
}

// Route mirrors routes table (migration 000006).
type Route struct {
	ID               string
	RouteName        string
	CreatedByStaffID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateRouteInput struct {
	StaffID   string
	RouteName string
}

type DeleteRouteInput struct {
	StaffID string
	RouteID string
}

type ListRoutesInput struct {
	CreatedByStaffID string
	Limit            int
	Offset           int
}

// StaffUsecase defines the inbound application port for staff operations.
// This interface is derived from the current DB schema (migrations/).
type StaffUsecase interface {
	// Auth / staff management
	Login(ctx context.Context, in LoginInput) (*LoginResult, error)
	CreateStaff(ctx context.Context, in CreateStaffInput) (*Staff, error)

	// Found items
	CreateFoundItem(ctx context.Context, in CreateFoundItemInput) (*FoundItem, error)
	UpdateFoundItemStatus(ctx context.Context, in UpdateFoundItemStatusInput) (*FoundItem, error)
	ListFoundItems(ctx context.Context, in ListFoundItemsInput) ([]FoundItem, error)
	SearchFoundItemMatchesByEmbedding(ctx context.Context, in SearchFoundItemMatchesByEmbeddingInput) ([]FoundItemMatch, error)

	// Claims
	ListClaims(ctx context.Context, in ListClaimsInput) ([]ItemClaim, error)
	ReviewClaim(ctx context.Context, in ReviewClaimInput) (*ItemClaim, error)

	// Routes
	CreateRoute(ctx context.Context, in CreateRouteInput) (*Route, error)
	DeleteRoute(ctx context.Context, in DeleteRouteInput) error
	ListRoutes(ctx context.Context, in ListRoutesInput) ([]Route, error)
}
