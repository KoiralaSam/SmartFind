package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	"smartfind/services/staff-service/internal/core/ports/inbound"
	"smartfind/shared/pgvector"
)

func (r *StaffRepository) SearchFoundItemMatchesByEmbedding(ctx context.Context, queryEmbedding []float32, limit int, minSimilarity float64) ([]inbound.FoundItemMatch, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("db pool is nil")
	}
	if len(queryEmbedding) != 1536 {
		return nil, errors.New("query_embedding must have 1536 dimensions")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	if minSimilarity < 0 {
		minSimilarity = 0
	}
	if minSimilarity > 1 {
		minSimilarity = 1
	}

	vecLit := pgvector.Literal(queryEmbedding)

	rows, err := r.pool.Query(ctx, `
		SELECT
			fi.id::text, fi.posted_by_staff_id::text,
			fi.item_name, COALESCE(fi.item_description, ''), fi.item_type, fi.brand, fi.model, fi.color, fi.material, fi.item_condition,
			COALESCE(fi.category, ''), COALESCE(fi.location_found, ''), COALESCE(fi.route_or_station, ''),
			COALESCE(fi.route_id::text, ''), fi.date_found,
			COALESCE(fi.image_keys, '{}'::text[]), COALESCE(fi.primary_image_key, ''),
			fi.status::text, fi.created_at, fi.updated_at,
			(1 - (fie.embedding <=> $1::vector))::float8 AS similarity
		FROM found_items fi
		JOIN found_item_embeddings fie ON fie.found_item_id = fi.id
		WHERE fi.status = 'unclaimed'
		  AND (1 - (fie.embedding <=> $1::vector)) >= $2
		ORDER BY fie.embedding <=> $1::vector
		LIMIT $3
	`, vecLit, minSimilarity, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]inbound.FoundItemMatch, 0)
	for rows.Next() {
		var it inbound.FoundItem
		var df pgtype.Date
		var imageKeys pgtype.FlatArray[string]
		var sim float64
		if err := rows.Scan(
			&it.ID, &it.PostedByStaffID,
			&it.ItemName, &it.ItemDescription, &it.ItemType, &it.Brand, &it.Model, &it.Color, &it.Material, &it.ItemCondition,
			&it.Category, &it.LocationFound, &it.RouteOrStation, &it.RouteID,
			&df, &imageKeys, &it.PrimaryImageKey, &it.Status, &it.CreatedAt, &it.UpdatedAt,
			&sim,
		); err != nil {
			return nil, err
		}
		if df.Valid {
			it.DateFound = df.Time
		}
		if imageKeys != nil {
			it.ImageKeys = []string(imageKeys)
		}
		out = append(out, inbound.FoundItemMatch{
			Item:            it,
			SimilarityScore: clamp01(sim),
		})
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
