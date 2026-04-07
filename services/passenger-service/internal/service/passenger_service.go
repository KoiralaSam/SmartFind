package service

import (
	"context"
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

func (s *PassengerService) Register(ctx context.Context, in inbound.RegisterInput) (*domain.Passenger, error) {
	_ = ctx
	now := time.Now()
	_ = s.repo

	return &domain.Passenger{
		ID:        "temp-id",
		Email:     in.Email,
		FullName:  in.Username, // maps to passengers.full_name in DB
		Phone:     "",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *PassengerService) Login(ctx context.Context, email string) (*domain.Passenger, error) {
	_ = ctx
	now := time.Now()
	return &domain.Passenger{
		ID:        "temp-id",
		Email:     email,
		FullName:  "Passenger",
		Phone:     "",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

