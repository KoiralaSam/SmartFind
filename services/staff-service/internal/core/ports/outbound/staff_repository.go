package outbound

import (
	"context"
	"errors"

	"smartfind/services/staff-service/internal/core/domain"
	"smartfind/services/staff-service/internal/core/ports/inbound"
)

// ErrStaffEmailExists is returned when inserting a duplicate staff email.
var ErrStaffEmailExists = errors.New("staff email already exists")

// ErrNotFound is returned when a row is missing or the operation cannot complete.
var ErrNotFound = errors.New("not found")

// ErrRouteNameExists is returned when creating a duplicate route name.
var ErrRouteNameExists = errors.New("route name already exists")

// StaffRepository defines persistence for staff accounts and staff-scoped data.
type StaffRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.Staff, error)
	Create(ctx context.Context, staff domain.Staff) (*domain.Staff, error)

	CreateFoundItem(ctx context.Context, in inbound.CreateFoundItemInput) (*inbound.FoundItem, error)
	DeleteFoundItem(ctx context.Context, foundItemID string) error
	UpsertFoundItemEmbedding(ctx context.Context, foundItemID string, embedding []float32) error
	UpdateFoundItemStatus(ctx context.Context, foundItemID, staffID, status string) (*inbound.FoundItem, error)
	ListFoundItems(ctx context.Context, in inbound.ListFoundItemsInput) ([]inbound.FoundItem, error)
	SearchFoundItemMatchesByEmbedding(ctx context.Context, queryEmbedding []float32, limit int, minSimilarity float64) ([]inbound.FoundItemMatch, error)

	ListClaims(ctx context.Context, in inbound.ListClaimsInput) ([]inbound.ItemClaim, error)
	UpdateClaimStatusForStaffItem(ctx context.Context, claimID, staffID, status string) (*inbound.ItemClaim, error)

	CreateRoute(ctx context.Context, staffID, routeName string) (*inbound.Route, error)
	DeleteRoute(ctx context.Context, routeID string) error
	ListRoutes(ctx context.Context, in inbound.ListRoutesInput) ([]inbound.Route, error)
}
