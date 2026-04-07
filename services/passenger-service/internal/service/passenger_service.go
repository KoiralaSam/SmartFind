package service

import (
	"context"
	"errors"
	"time"

	"smartfind/services/passenger-service/internal/core/domain"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/services/passenger-service/internal/core/ports/outbound"
)

// PassengerService implements the inbound PassengerUsecase port.
type PassengerService struct {
	repo outbound.PassengerRepository
}

func NewPassengerService(repo outbound.PassengerRepository) inbound.PassengerUsecase {
	return &PassengerService{repo: repo}
}

func (s *PassengerService) Login(ctx context.Context, email string) (*domain.Passenger, error) {
	_ = ctx
	now := time.Now()
	return &domain.Passenger{
		ID:        "temp-id",
		Email:     email,
		FullName:  "Passenger",
		Phone:     "",
		AvatarURL: "",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *PassengerService) CreateLostReport(ctx context.Context, in inbound.CreateLostReportInput) (*inbound.LostReport, error) {
	_ = ctx
	_ = in
	return nil, errors.New("CreateLostReport not implemented")
}

func (s *PassengerService) ListLostReports(ctx context.Context, in inbound.ListLostReportsInput) ([]inbound.LostReport, error) {
	_ = ctx
	_ = in
	return nil, errors.New("ListLostReports not implemented")
}

func (s *PassengerService) DeleteLostReport(ctx context.Context, passengerID, lostReportID string) error {
	_ = ctx
	_ = passengerID
	_ = lostReportID
	return errors.New("DeleteLostReport not implemented")
}

func (s *PassengerService) SearchFoundItemMatches(ctx context.Context, in inbound.SearchFoundItemsInput) ([]inbound.FoundItemMatch, error) {
	_ = ctx
	_ = in
	return nil, errors.New("SearchFoundItemMatches not implemented")
}

func (s *PassengerService) FileClaim(ctx context.Context, in inbound.FileClaimInput) (*inbound.ItemClaim, error) {
	_ = ctx
	_ = in
	return nil, errors.New("FileClaim not implemented")
}

