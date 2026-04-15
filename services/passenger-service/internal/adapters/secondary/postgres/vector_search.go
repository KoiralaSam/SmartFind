package postgres

import (
	"context"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
)

func (r *PassengerRepository) searchFoundItemMatchesVector(ctx context.Context, passengerID string, lostReportID string, limit int) ([]inbound.FoundItemMatch, error) {
	// If found item embeddings are not populated yet, skip vector search quickly.
	var hasAny bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM found_item_embeddings LIMIT 1)`).Scan(&hasAny); err != nil {
		return nil, err
	}
	if !hasAny {
		return []inbound.FoundItemMatch{}, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			fi.id::text,
			fi.item_name, fi.item_description, fi.item_type, fi.brand, fi.model, fi.color, fi.material, fi.item_condition,
			fi.category, fi.location_found, fi.route_or_station, COALESCE(fi.route_id::text, ''), fi.date_found,
			fi.status,
			(1 - (fie.embedding <=> lre.embedding))::float8 AS similarity
		FROM found_items fi
		JOIN found_item_embeddings fie ON fie.found_item_id = fi.id
		JOIN lost_report_embeddings lre ON lre.lost_report_id = NULLIF($1, '')::uuid
		JOIN lost_reports lr ON lr.id = lre.lost_report_id AND lr.reporter_passenger_id = NULLIF($2, '')::uuid
		WHERE fi.status = 'unclaimed'
		ORDER BY fie.embedding <=> lre.embedding
		LIMIT $3
	`, lostReportID, passengerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]inbound.FoundItemMatch, 0)
	for rows.Next() {
		var m inbound.FoundItemMatch
		var routeID string
		var similarity float64
		if scanErr := rows.Scan(
			&m.FoundItemID,
			&m.ItemName, &m.ItemDescription, &m.ItemType, &m.Brand, &m.Model, &m.Color, &m.Material, &m.ItemCondition,
			&m.Category, &m.LocationFound, &m.RouteOrStation, &routeID, &m.DateFound,
			&m.Status,
			&similarity,
		); scanErr != nil {
			return nil, scanErr
		}
		m.RouteID = routeID
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
