package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
)

func (r *PassengerRepository) CreateItemClaimAndMarkLostReportMatched(ctx context.Context, claim inbound.ItemClaim) (*inbound.ItemClaim, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("db pool is nil")
	}

	if claim.CreatedAt.IsZero() {
		claim.CreatedAt = time.Now()
	}
	if claim.UpdatedAt.IsZero() {
		claim.UpdatedAt = claim.CreatedAt
	}
	if claim.Status == "" {
		claim.Status = "pending"
	}

	passengerID := strings.TrimSpace(claim.ClaimantPassengerID)
	lostReportID := strings.TrimSpace(claim.LostReportID)
	if passengerID == "" || lostReportID == "" {
		return nil, errors.New("passenger_id and lost_report_id are required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO item_claims (
			item_id, claimant_passenger_id, lost_report_id,
			message, status, created_at, updated_at
		) VALUES (
			$1, $2, NULLIF($3, '')::uuid,
			$4, $5, $6, $7
		)
		RETURNING id::text, created_at, updated_at
	`, claim.ItemID, claim.ClaimantPassengerID, claim.LostReportID,
		claim.Message, claim.Status, claim.CreatedAt, claim.UpdatedAt,
	).Scan(&claim.ID, &claim.CreatedAt, &claim.UpdatedAt)
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `
		UPDATE lost_reports
		SET status = 'matched', updated_at = NOW()
		WHERE id = $1::uuid AND reporter_passenger_id = $2::uuid
	`, lostReportID, passengerID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, errors.New("lost report not found")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &claim, nil
}

var _ = pgx.ErrNoRows
