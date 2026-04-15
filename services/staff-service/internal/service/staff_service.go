package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"smartfind/services/staff-service/internal/core/domain"
	"smartfind/services/staff-service/internal/core/ports/inbound"
	"smartfind/services/staff-service/internal/core/ports/outbound"
	"smartfind/shared/env"
	"smartfind/shared/jwt"
)

var foundItemStatuses = map[string]struct{}{
	"unclaimed": {},
	"claimed":   {},
	"returned":  {},
	"archived":  {},
}

// Default transit codes apply when TRANSIT_CODE is unset (local dev); comma-separated
// values are allowed. Login/session tokens use JWT_SECRET via shared/jwt.
const defaultTransitCodes = "SMARTFIND-TRANSIT-2026,DEMO-INVITE"

// StaffService implements inbound.StaffUsecase.
type StaffService struct {
	repo outbound.StaffRepository
}

// NewStaffService wires the staff use case.
func NewStaffService(repo outbound.StaffRepository) inbound.StaffUsecase {
	return &StaffService{repo: repo}
}

// Login checks email/password and issues a session JWT (JWT_SECRET / JWT_TTL_SECONDS from env).
func (s *StaffService) Login(ctx context.Context, in inbound.LoginInput) (*inbound.LoginResult, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" || strings.TrimSpace(in.Password) == "" {
		return nil, errors.New("email and password are required")
	}

	record, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if record == nil || strings.TrimSpace(record.PasswordHash) == "" {
		return nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(in.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := jwt.GenerateUserToken(record.ID, record.Email)
	if err != nil {
		return nil, err
	}

	return &inbound.LoginResult{
		Staff:        toInboundStaff(record),
		SessionToken: token,
	}, nil
}

// CreateStaff validates the signup value against TRANSIT_CODE (comma-separated allowed in env).
func (s *StaffService) CreateStaff(ctx context.Context, in inbound.CreateStaffInput) (*inbound.Staff, error) {
	if !validTransitCode(in.TransitCode) {
		return nil, errors.New("invalid transit code")
	}

	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if len(strings.TrimSpace(in.Password)) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, outbound.ErrStaffEmailExists
	}

	fullName := strings.TrimSpace(in.FullName)
	if fullName == "" {
		if at := strings.IndexByte(email, '@'); at > 0 {
			fullName = email[:at]
		} else {
			fullName = "Staff"
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, domain.Staff{
		FullName:     fullName,
		Email:        email,
		PasswordHash: string(hash),
	})
	if err != nil {
		return nil, err
	}

	return toInboundStaff(created), nil
}

func validTransitCode(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	raw := env.GetString("TRANSIT_CODE", defaultTransitCodes)
	for _, part := range strings.Split(raw, ",") {
		if strings.TrimSpace(part) == code {
			return true
		}
	}
	return false
}

func toInboundStaff(d *domain.Staff) *inbound.Staff {
	if d == nil {
		return nil
	}
	return &inbound.Staff{
		ID:        d.ID,
		FullName:  d.FullName,
		Email:     d.Email,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func (s *StaffService) CreateFoundItem(ctx context.Context, in inbound.CreateFoundItemInput) (*inbound.FoundItem, error) {
	if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.ItemName) == "" {
		return nil, errors.New("staff_id and item_name are required")
	}
	return s.repo.CreateFoundItem(ctx, in)
}

func (s *StaffService) UpdateFoundItemStatus(ctx context.Context, in inbound.UpdateFoundItemStatusInput) (*inbound.FoundItem, error) {
	if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.FoundItemID) == "" {
		return nil, errors.New("staff_id and found_item_id are required")
	}
	st := strings.ToLower(strings.TrimSpace(in.Status))
	if _, ok := foundItemStatuses[st]; !ok {
		return nil, errors.New("invalid status: must be unclaimed, claimed, returned, or archived")
	}
	item, err := s.repo.UpdateFoundItemStatus(ctx, in.FoundItemID, in.StaffID, st)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *StaffService) ListFoundItems(ctx context.Context, in inbound.ListFoundItemsInput) ([]inbound.FoundItem, error) {
	in2 := in
	if strings.TrimSpace(in.Status) != "" {
		st := strings.ToLower(strings.TrimSpace(in.Status))
		if _, ok := foundItemStatuses[st]; !ok {
			return nil, errors.New("invalid status filter")
		}
		in2.Status = st
	}
	return s.repo.ListFoundItems(ctx, in2)
}

func (s *StaffService) ListClaims(ctx context.Context, in inbound.ListClaimsInput) ([]inbound.ItemClaim, error) {
	in2 := in
	if strings.TrimSpace(in.Status) != "" {
		st := strings.ToLower(strings.TrimSpace(in.Status))
		if !validClaimStatusFilter(st) {
			return nil, errors.New("invalid status filter")
		}
		in2.Status = st
	}
	return s.repo.ListClaims(ctx, in2)
}

func (s *StaffService) ReviewClaim(ctx context.Context, in inbound.ReviewClaimInput) (*inbound.ItemClaim, error) {
	if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.ClaimID) == "" {
		return nil, errors.New("staff_id and claim_id are required")
	}
	d := strings.ToLower(strings.TrimSpace(in.Decision))
	if d != "approved" && d != "rejected" {
		return nil, errors.New("decision must be approved or rejected")
	}
	return s.repo.UpdateClaimStatusForStaffItem(ctx, in.ClaimID, in.StaffID, d)
}

func (s *StaffService) CreateRoute(ctx context.Context, in inbound.CreateRouteInput) (*inbound.Route, error) {
	if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.RouteName) == "" {
		return nil, errors.New("staff_id and route_name are required")
	}
	return s.repo.CreateRoute(ctx, in.StaffID, strings.TrimSpace(in.RouteName))
}

func (s *StaffService) DeleteRoute(ctx context.Context, in inbound.DeleteRouteInput) error {
	if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.RouteID) == "" {
		return errors.New("staff_id and route_id are required")
	}
	return s.repo.DeleteRouteIfOwner(ctx, in.StaffID, in.RouteID)
}

func (s *StaffService) ListRoutes(ctx context.Context, in inbound.ListRoutesInput) ([]inbound.Route, error) {
	return s.repo.ListRoutes(ctx, in)
}

func validClaimStatusFilter(s string) bool {
	switch s {
	case "pending", "approved", "rejected", "cancelled":
		return true
	default:
		return false
	}
}
