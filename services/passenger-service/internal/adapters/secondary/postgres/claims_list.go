package postgres

import (
	"context"
	"strconv"
	"strings"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
)

func (r *PassengerRepository) ListMyClaims(ctx context.Context, passengerID string, status string, limit int, offset int) ([]inbound.ItemClaim, error) {
	passengerID = strings.TrimSpace(passengerID)
	if passengerID == "" {
		return []inbound.ItemClaim{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	q := strings.Builder{}
	q.WriteString(`
		SELECT id::text, item_id::text, claimant_passenger_id::text, COALESCE(lost_report_id::text, ''),
			COALESCE(message, ''), status::text, created_at, updated_at
		FROM item_claims
		WHERE claimant_passenger_id = $1::uuid
	`)
	args := []any{passengerID}
	n := 2
	if strings.TrimSpace(status) != "" {
		q.WriteString(` AND status = $` + strconv.Itoa(n) + `::claim_status`)
		args = append(args, status)
		n++
	}
	q.WriteString(` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1))
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]inbound.ItemClaim, 0)
	for rows.Next() {
		var c inbound.ItemClaim
		if err := rows.Scan(
			&c.ID, &c.ItemID, &c.ClaimantPassengerID, &c.LostReportID,
			&c.Message, &c.Status, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}
