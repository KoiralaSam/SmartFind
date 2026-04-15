package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"smartfind/services/passenger-service/internal/core/domain"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/services/passenger-service/internal/core/ports/outbound"
	"smartfind/shared/env"
	"smartfind/shared/googleauth"
	"smartfind/shared/jwt"
)

// PassengerService implements the inbound PassengerUsecase port.
type PassengerService struct {
	repo outbound.PassengerRepository
}

// NewPassengerService wires the passenger use case.
// JWT creation/verification reads JWT_SECRET (and optional JWT_TTL_SECONDS) from the environment.
func NewPassengerService(repo outbound.PassengerRepository) inbound.PassengerUsecase {
	return &PassengerService{repo: repo}
}

func (s *PassengerService) Login(ctx context.Context, in inbound.LoginInput) (*inbound.LoginResult, error) {
	if strings.TrimSpace(in.IDToken) == "" {
		return nil, errors.New("id_token is required")
	}

	clientID := env.GetString("GOOGLE_CLIENT_ID", "")
	profile, err := googleauth.VerifyIDToken(ctx, in.IDToken, clientID)
	if err != nil {
		return nil, err
	}

	email := profile.Email
	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var p *domain.Passenger
	if existing == nil {
		created, err := s.repo.Create(ctx, domain.Passenger{
			Email:     email,
			FullName:  profile.FullName,
			Phone:     "",
			AvatarURL: profile.PictureURL,
		})
		if err != nil {
			return nil, err
		}
		p = created
	} else {
		p = existing
		if profile.FullName != p.FullName || profile.PictureURL != p.AvatarURL {
			p.FullName = profile.FullName
			p.AvatarURL = profile.PictureURL
			if err := s.repo.Update(ctx, *p); err != nil {
				return nil, err
			}
		}
	}

	token, err := jwt.GenerateUserToken(p.ID, p.Email)
	if err != nil {
		return nil, err
	}
	return &inbound.LoginResult{Passenger: p, SessionToken: token}, nil
}

func (s *PassengerService) CreateLostReport(ctx context.Context, in inbound.CreateLostReportInput) (*inbound.LostReport, error) {
	embeddingText := buildLostReportEmbeddingText(in)
	embedding, err := embedTextOpenAI(ctx, embeddingText)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	rpt := inbound.LostReport{
		ReporterPassengerID: in.PassengerID,
		ItemName:            in.ItemName,
		ItemDescription:     in.ItemDescription,
		ItemType:            in.ItemType,
		Brand:               in.Brand,
		Model:               in.Model,
		Color:               in.Color,
		Material:            in.Material,
		ItemCondition:       in.ItemCondition,
		Category:            in.Category,
		LocationLost:        in.LocationLost,
		RouteOrStation:      in.RouteOrStation,
		RouteID:             in.RouteID,
		DateLost:            in.DateLost,
		Status:              "open",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	created, err := s.repo.CreateLostReport(ctx, rpt)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpsertLostReportEmbedding(ctx, created.ID, embedding); err != nil {
		_ = s.repo.DeleteLostReport(ctx, in.PassengerID, created.ID)
		return nil, err
	}
	return created, nil
}

func (s *PassengerService) ListLostReports(ctx context.Context, in inbound.ListLostReportsInput) ([]inbound.LostReport, error) {
	reports, err := s.repo.ListLostReports(ctx, in.PassengerID, in.Status)
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func (s *PassengerService) DeleteLostReport(ctx context.Context, passengerID, lostReportID string) error {
	return s.repo.DeleteLostReport(ctx, passengerID, lostReportID)
}

func (s *PassengerService) SearchFoundItemMatches(ctx context.Context, in inbound.SearchFoundItemsInput) ([]inbound.FoundItemMatch, error) {
	matches, err := s.repo.SearchFoundItemMatches(ctx, in.PassengerID, in.LostReportID, in.Limit)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func (s *PassengerService) FileClaim(ctx context.Context, in inbound.FileClaimInput) (*inbound.ItemClaim, error) {
	now := time.Now()
	claim := inbound.ItemClaim{
		ItemID:              in.FoundItemID,
		ClaimantPassengerID: in.PassengerID,
		LostReportID:        in.LostReportID,
		Message:             in.Message,
		Status:              "pending",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	created, err := s.repo.CreateItemClaim(ctx, claim)
	if err != nil {
		return nil, err
	}
	return created, nil
}
