package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"smartfind/services/passenger-service/internal/core/domain"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/services/passenger-service/internal/core/ports/outbound"
	"smartfind/shared/env"
	"smartfind/shared/googleauth"
	"smartfind/shared/jwt"
	"smartfind/shared/util"

	staffpb "smartfind/shared/proto/staff"
)

// PassengerService implements the inbound PassengerUsecase port.
type PassengerService struct {
	repo        outbound.PassengerRepository
	staffClient staffpb.StaffServiceClient
}

// NewPassengerService wires the passenger use case.
// JWT creation/verification reads JWT_SECRET (and optional JWT_TTL_SECONDS) from the environment.
func NewPassengerService(repo outbound.PassengerRepository, staffClient staffpb.StaffServiceClient) inbound.PassengerUsecase {
	return &PassengerService{repo: repo, staffClient: staffClient}
}

// resolvePassengerAvatar prefers the Google profile picture; otherwise keeps a
// non-empty stored avatar, or assigns a stable random Lego avatar (indices 0–9).
func resolvePassengerAvatar(googlePicture string, existingAvatar string) string {
	if s := strings.TrimSpace(googlePicture); s != "" {
		return s
	}
	if s := strings.TrimSpace(existingAvatar); s != "" {
		return s
	}
	return util.GetRandomAvatar(rand.IntN(10))
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
		avatarURL := resolvePassengerAvatar(profile.PictureURL, "")
		created, err := s.repo.Create(ctx, domain.Passenger{
			Email:     email,
			FullName:  profile.FullName,
			Phone:     "",
			AvatarURL: avatarURL,
		})
		if err != nil {
			return nil, err
		}
		p = created
	} else {
		p = existing
		nextAvatar := resolvePassengerAvatar(profile.PictureURL, p.AvatarURL)
		if profile.FullName != p.FullName || nextAvatar != p.AvatarURL {
			p.FullName = profile.FullName
			p.AvatarURL = nextAvatar
			if err := s.repo.Update(ctx, *p); err != nil {
				return nil, err
			}
		}
	}

	token, err := jwt.GeneratePassengerToken(p.ID, p.Email)
	if err != nil {
		return nil, err
	}
	return &inbound.LoginResult{Passenger: p, SessionToken: token}, nil
}

func (s *PassengerService) CreateLostReport(ctx context.Context, in inbound.CreateLostReportInput) (*inbound.LostReport, error) {
	embedding, err := embedLostReportOpenAI(ctx, in)
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
	if s.staffClient == nil {
		return nil, errors.New("staff service client is not configured")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}

	embedding, err := s.repo.GetLostReportEmbeddingForPassenger(ctx, in.PassengerID, in.LostReportID)
	if err != nil {
		return nil, err
	}
	if len(embedding) == 0 {
		return []inbound.FoundItemMatch{}, nil
	}

	minSim := env.GetFloat("MATCH_MIN_SIMILARITY", 0.82)
	if minSim < 0 {
		minSim = 0
	}
	if minSim > 1 {
		minSim = 1
	}

	resp, err := s.staffClient.SearchFoundItemMatchesByEmbedding(ctx, &staffpb.SearchFoundItemMatchesByEmbeddingRequest{
		QueryEmbedding: embedding,
		Limit:          int32(limit),
		MinSimilarity:  minSim,
	})
	if err != nil {
		return nil, err
	}

	out := make([]inbound.FoundItemMatch, 0, len(resp.GetMatches()))
	for _, m := range resp.GetMatches() {
		if m == nil || m.GetItem() == nil {
			continue
		}
		it := m.GetItem()
		var dt time.Time
		if it.GetDateFound() != nil {
			dt = it.GetDateFound().AsTime()
		}
		out = append(out, inbound.FoundItemMatch{
			FoundItemID:     it.GetId(),
			ItemName:        it.GetItemName(),
			ItemDescription: it.GetItemDescription(),
			ItemType:        it.GetItemType(),
			Brand:           it.GetBrand(),
			Model:           it.GetModel(),
			Color:           it.GetColor(),
			Material:        it.GetMaterial(),
			ItemCondition:   it.GetItemCondition(),
			Category:        it.GetCategory(),
			LocationFound:   it.GetLocationFound(),
			RouteOrStation:  it.GetRouteOrStation(),
			RouteID:         it.GetRouteId(),
			DateFound:       dt,
			Status:          it.GetStatus(),
			SimilarityScore: m.GetSimilarityScore(),
			ImageURLs:       it.GetImageUrls(),
			PrimaryImageURL: it.GetPrimaryImageUrl(),
		})
	}
	return out, nil
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
	created, err := s.repo.CreateItemClaimAndMarkLostReportMatched(ctx, claim)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *PassengerService) ListNotifications(ctx context.Context, in inbound.ListNotificationsInput) ([]inbound.PassengerMatchNotification, error) {
	return s.repo.ListNotifications(ctx, in.PassengerID, in.Limit, in.UnreadOnly, in.CreatedBefore)
}

func (s *PassengerService) MarkNotificationRead(ctx context.Context, in inbound.MarkNotificationReadInput) error {
	return s.repo.MarkNotificationsRead(ctx, in.PassengerID, in.NotificationIDs)
}
