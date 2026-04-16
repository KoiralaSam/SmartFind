package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"smartfind/shared/pgvector"
)

func (r *PassengerRepository) GetLostReportEmbeddingForPassenger(ctx context.Context, passengerID string, lostReportID string) ([]float32, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("db pool is nil")
	}
	passengerID = strings.TrimSpace(passengerID)
	lostReportID = strings.TrimSpace(lostReportID)
	if passengerID == "" || lostReportID == "" {
		return nil, errors.New("passengerID and lostReportID are required")
	}

	var lit string
	err := r.pool.QueryRow(ctx, `
		SELECT lre.embedding::text
		FROM lost_report_embeddings lre
		JOIN lost_reports lr ON lr.id = lre.lost_report_id
		WHERE lre.lost_report_id = $1::uuid
		  AND lr.reporter_passenger_id = $2::uuid
	`, lostReportID, passengerID).Scan(&lit)
	if errors.Is(err, pgx.ErrNoRows) {
		return []float32{}, nil
	}
	if err != nil {
		return nil, err
	}

	vec, err := pgvector.ParseLiteral(lit)
	if err != nil {
		return nil, err
	}
	if len(vec) != 1536 {
		return nil, errors.New("embedding must have 1536 dimensions")
	}
	return vec, nil
}
