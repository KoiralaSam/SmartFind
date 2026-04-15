package pgvector

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FoundItemMatch struct {
	FoundItemID     string
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
	SimilarityScore float64
}

// SearchFoundItemMatches returns the most similar found items for a lost report using pgvector.
//
// - If passengerID is non-empty, it enforces that the lost report belongs to that passenger.
// - Returns an empty slice if embeddings are not present/populated.
func SearchFoundItemMatches(ctx context.Context, pool *pgxpool.Pool, lostReportID string, passengerID string, limit int) ([]FoundItemMatch, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	if limit <= 0 {
		limit = 10
	}

	var hasAny bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM found_item_embeddings LIMIT 1)`).Scan(&hasAny); err != nil {
		return nil, err
	}
	if !hasAny {
		return []FoundItemMatch{}, nil
	}

	rows, err := pool.Query(ctx, `
		SELECT
			fi.id::text,
			fi.item_name, fi.item_description, fi.item_type, fi.brand, fi.model, fi.color, fi.material, fi.item_condition,
			fi.category, fi.location_found, fi.route_or_station, COALESCE(fi.route_id::text, ''), fi.date_found,
			fi.status,
			(1 - (fie.embedding <=> lre.embedding))::float8 AS similarity
		FROM found_items fi
		JOIN found_item_embeddings fie ON fie.found_item_id = fi.id
		JOIN lost_report_embeddings lre ON lre.lost_report_id = NULLIF($1, '')::uuid
		JOIN lost_reports lr ON lr.id = lre.lost_report_id
		WHERE fi.status = 'unclaimed'
		  AND (NULLIF($2, '') IS NULL OR lr.reporter_passenger_id = NULLIF($2, '')::uuid)
		ORDER BY fie.embedding <=> lre.embedding
		LIMIT $3
	`, lostReportID, passengerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]FoundItemMatch, 0)
	for rows.Next() {
		var m FoundItemMatch
		var similarity float64
		if scanErr := rows.Scan(
			&m.FoundItemID,
			&m.ItemName, &m.ItemDescription, &m.ItemType, &m.Brand, &m.Model, &m.Color, &m.Material, &m.ItemCondition,
			&m.Category, &m.LocationFound, &m.RouteOrStation, &m.RouteID, &m.DateFound,
			&m.Status,
			&similarity,
		); scanErr != nil {
			return nil, scanErr
		}
		m.SimilarityScore = clamp01(similarity)
		out = append(out, m)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
