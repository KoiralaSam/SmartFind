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

func (s *PassengerService) GetPassengerByID(ctx context.Context, passengerID string) (*domain.Passenger, error) {
	passengerID = strings.TrimSpace(passengerID)
	if passengerID == "" {
		return nil, errors.New("passenger_id is required")
	}
	return s.repo.GetByID(ctx, passengerID)
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

// UpdateLostReport applies the non-nil fields of `in` to the passenger's
// lost report. When any of the twelve slots that feed the match embedding
// change, the OpenAI embedding is recomputed and upserted so subsequent
// similarity searches reflect the edit. If the embedding call fails the
// database edit is kept but the error is swallowed (the old vector stays
// in place) so an outage of the embeddings provider does not block edits.
func (s *PassengerService) UpdateLostReport(ctx context.Context, in inbound.UpdateLostReportInput) (*inbound.LostReport, error) {
	before, err := s.repo.GetLostReportForPassenger(ctx, in.PassengerID, in.LostReportID)
	if err != nil {
		return nil, err
	}
	if before == nil {
		return nil, outbound.ErrLostReportNotFound
	}

	updated, err := s.repo.UpdateLostReport(ctx, in)
	if err != nil {
		return nil, err
	}

	if lostReportEmbeddingChanged(before, updated) {
		emb, embErr := embedLostReportOpenAI(ctx, inbound.CreateLostReportInput{
			PassengerID:     updated.ReporterPassengerID,
			ItemName:        updated.ItemName,
			ItemDescription: updated.ItemDescription,
			ItemType:        updated.ItemType,
			Brand:           updated.Brand,
			Model:           updated.Model,
			Color:           updated.Color,
			Material:        updated.Material,
			ItemCondition:   updated.ItemCondition,
			Category:        updated.Category,
			LocationLost:    updated.LocationLost,
			RouteOrStation:  updated.RouteOrStation,
			RouteID:         updated.RouteID,
			DateLost:        updated.DateLost,
		})
		if embErr == nil {
			_ = s.repo.UpsertLostReportEmbedding(ctx, updated.ID, emb)
		}
	}

	return updated, nil
}

// lostReportEmbeddingChanged returns true when any of the twelve slots that
// buildLostReportEmbeddingText reads have changed between the pre- and
// post-update rows. Fields that do not feed the embedding (status,
// timestamps) are ignored.
func lostReportEmbeddingChanged(a, b *inbound.LostReport) bool {
	if a == nil || b == nil {
		return true
	}
	return a.ItemName != b.ItemName ||
		a.ItemDescription != b.ItemDescription ||
		a.ItemType != b.ItemType ||
		a.Brand != b.Brand ||
		a.Model != b.Model ||
		a.Color != b.Color ||
		a.Material != b.Material ||
		a.ItemCondition != b.ItemCondition ||
		a.Category != b.Category ||
		a.LocationLost != b.LocationLost ||
		a.RouteOrStation != b.RouteOrStation ||
		a.RouteID != b.RouteID
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
		// Backfill embeddings for older lost reports created before embeddings were introduced.
		rpt, err := s.repo.GetLostReportForPassenger(ctx, in.PassengerID, in.LostReportID)
		if err != nil {
			return nil, err
		}
		if rpt == nil {
			return []inbound.FoundItemMatch{}, nil
		}
		emb, err := embedLostReportOpenAI(ctx, inbound.CreateLostReportInput{
			PassengerID:     rpt.ReporterPassengerID,
			ItemName:        rpt.ItemName,
			ItemDescription: rpt.ItemDescription,
			ItemType:        rpt.ItemType,
			Brand:           rpt.Brand,
			Model:           rpt.Model,
			Color:           rpt.Color,
			Material:        rpt.Material,
			ItemCondition:   rpt.ItemCondition,
			Category:        rpt.Category,
			LocationLost:    rpt.LocationLost,
			RouteOrStation:  rpt.RouteOrStation,
			RouteID:         rpt.RouteID,
			DateLost:        rpt.DateLost,
		})
		if err != nil {
			// If embeddings aren't available (e.g. missing API key), degrade gracefully.
			return []inbound.FoundItemMatch{}, nil
		}
		if err := s.repo.UpsertLostReportEmbedding(ctx, rpt.ID, emb); err != nil {
			return nil, err
		}
		embedding = emb
	}
	if len(embedding) == 0 {
		return []inbound.FoundItemMatch{}, nil
	}

	minSim := env.GetFloat("MATCH_MIN_SIMILARITY", 0.65)
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

func (s *PassengerService) ListMyClaims(ctx context.Context, in inbound.ListMyClaimsInput) ([]inbound.ItemClaim, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListMyClaims(ctx, in.PassengerID, in.Status, limit, offset)
}
