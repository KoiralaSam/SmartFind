package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"smartfind/services/staff-service/internal/core/ports/inbound"
	"smartfind/services/staff-service/internal/core/ports/outbound"
)

func (r *StaffRepository) CreateFoundItem(ctx context.Context, in inbound.CreateFoundItemInput) (*inbound.FoundItem, error) {
	now := time.Now()

	var dateArg any
	if in.DateFound.IsZero() {
		dateArg = nil
	} else {
		dateArg = in.DateFound
	}

	var it inbound.FoundItem
	var df pgtype.Date
	err := r.pool.QueryRow(ctx, `
		INSERT INTO found_items (
			posted_by_staff_id,
			item_name, item_description, item_type, brand, model, color, material, item_condition,
			category, location_found, route_or_station, route_id, date_found,
			status, created_at, updated_at
		) VALUES (
			$1::uuid,
			$2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, NULLIF($13, '')::uuid, $14,
			'unclaimed', $15, $16
		)
		RETURNING
			id::text, posted_by_staff_id::text,
			item_name, COALESCE(item_description, ''), item_type, brand, model, color, material, item_condition,
			COALESCE(category, ''), COALESCE(location_found, ''), COALESCE(route_or_station, ''),
			COALESCE(route_id::text, ''), date_found,
			status::text, created_at, updated_at
	`, in.StaffID,
		in.ItemName, in.ItemDescription, in.ItemType, in.Brand, in.Model, in.Color, in.Material, in.ItemCondition,
		"", in.LocationFound, in.RouteOrStation, in.RouteID, dateArg,
		now, now,
	).Scan(
		&it.ID, &it.PostedByStaffID,
		&it.ItemName, &it.ItemDescription, &it.ItemType, &it.Brand, &it.Model, &it.Color, &it.Material, &it.ItemCondition,
		&it.Category, &it.LocationFound, &it.RouteOrStation, &it.RouteID,
		&df, &it.Status, &it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if df.Valid {
		it.DateFound = df.Time
	}
	return &it, nil
}

func (r *StaffRepository) UpdateFoundItemStatus(ctx context.Context, foundItemID, staffID, status string) (*inbound.FoundItem, error) {
	var it inbound.FoundItem
	var df pgtype.Date
	err := r.pool.QueryRow(ctx, `
		UPDATE found_items
		SET status = $1::found_item_status, updated_at = NOW()
		WHERE id = $2::uuid AND posted_by_staff_id = $3::uuid
		RETURNING
			id::text, posted_by_staff_id::text,
			item_name, COALESCE(item_description, ''), item_type, brand, model, color, material, item_condition,
			COALESCE(category, ''), COALESCE(location_found, ''), COALESCE(route_or_station, ''),
			COALESCE(route_id::text, ''), date_found,
			status::text, created_at, updated_at
	`, status, foundItemID, staffID,
	).Scan(
		&it.ID, &it.PostedByStaffID,
		&it.ItemName, &it.ItemDescription, &it.ItemType, &it.Brand, &it.Model, &it.Color, &it.Material, &it.ItemCondition,
		&it.Category, &it.LocationFound, &it.RouteOrStation, &it.RouteID,
		&df, &it.Status, &it.CreatedAt, &it.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, outbound.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if df.Valid {
		it.DateFound = df.Time
	}
	return &it, nil
}

func (r *StaffRepository) ListFoundItems(ctx context.Context, in inbound.ListFoundItemsInput) ([]inbound.FoundItem, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	q := strings.Builder{}
	q.WriteString(`
		SELECT
			id::text, posted_by_staff_id::text,
			item_name, COALESCE(item_description, ''), item_type, brand, model, color, material, item_condition,
			COALESCE(category, ''), COALESCE(location_found, ''), COALESCE(route_or_station, ''),
			COALESCE(route_id::text, ''), date_found,
			status::text, created_at, updated_at
		FROM found_items
		WHERE 1=1
	`)
	args := []any{}
	n := 1
	if strings.TrimSpace(in.Status) != "" {
		q.WriteString(` AND status = $` + strconv.Itoa(n) + `::found_item_status`)
		args = append(args, in.Status)
		n++
	}
	if strings.TrimSpace(in.RouteID) != "" {
		q.WriteString(` AND route_id = $` + strconv.Itoa(n) + `::uuid`)
		args = append(args, in.RouteID)
		n++
	}
	if strings.TrimSpace(in.PostedByStaffID) != "" {
		q.WriteString(` AND posted_by_staff_id = $` + strconv.Itoa(n) + `::uuid`)
		args = append(args, in.PostedByStaffID)
		n++
	}
	q.WriteString(` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1))
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFoundItemRows(rows)
}

func scanFoundItemRows(rows pgx.Rows) ([]inbound.FoundItem, error) {
	out := make([]inbound.FoundItem, 0)
	for rows.Next() {
		var it inbound.FoundItem
		var df pgtype.Date
		if err := rows.Scan(
			&it.ID, &it.PostedByStaffID,
			&it.ItemName, &it.ItemDescription, &it.ItemType, &it.Brand, &it.Model, &it.Color, &it.Material, &it.ItemCondition,
			&it.Category, &it.LocationFound, &it.RouteOrStation, &it.RouteID,
			&df, &it.Status, &it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if df.Valid {
			it.DateFound = df.Time
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *StaffRepository) ListClaims(ctx context.Context, in inbound.ListClaimsInput) ([]inbound.ItemClaim, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	q := strings.Builder{}
	q.WriteString(`
		SELECT id::text, item_id::text, claimant_passenger_id::text, COALESCE(lost_report_id::text, ''),
			COALESCE(message, ''), status::text, created_at, updated_at
		FROM item_claims
		WHERE 1=1
	`)
	args := []any{}
	n := 1
	if strings.TrimSpace(in.Status) != "" {
		q.WriteString(` AND status = $` + strconv.Itoa(n) + `::claim_status`)
		args = append(args, in.Status)
		n++
	}
	if strings.TrimSpace(in.ItemID) != "" {
		q.WriteString(` AND item_id = $` + strconv.Itoa(n) + `::uuid`)
		args = append(args, in.ItemID)
		n++
	}
	if strings.TrimSpace(in.PassengerID) != "" {
		q.WriteString(` AND claimant_passenger_id = $` + strconv.Itoa(n) + `::uuid`)
		args = append(args, in.PassengerID)
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
			&c.ID, &c.ItemID, &c.ClaimantPassengerID, &c.LostReportID, &c.Message, &c.Status, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *StaffRepository) UpdateClaimStatusForStaffItem(ctx context.Context, claimID, staffID, status string) (*inbound.ItemClaim, error) {
	var c inbound.ItemClaim
	err := r.pool.QueryRow(ctx, `
		UPDATE item_claims AS c
		SET status = $1::claim_status, updated_at = NOW()
		FROM found_items AS fi
		WHERE c.id = $2::uuid
			AND c.item_id = fi.id
			AND fi.posted_by_staff_id = $3::uuid
			AND c.status = 'pending'::claim_status
		RETURNING c.id::text, c.item_id::text, c.claimant_passenger_id::text, COALESCE(c.lost_report_id::text, ''),
			COALESCE(c.message, ''), c.status::text, c.created_at, c.updated_at
	`, status, claimID, staffID,
	).Scan(
		&c.ID, &c.ItemID, &c.ClaimantPassengerID, &c.LostReportID, &c.Message, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, outbound.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *StaffRepository) CreateRoute(ctx context.Context, staffID, routeName string) (*inbound.Route, error) {
	routeName = strings.TrimSpace(routeName)
	if routeName == "" {
		return nil, errors.New("route_name is required")
	}
	var rt inbound.Route
	err := r.pool.QueryRow(ctx, `
		INSERT INTO routes (id, route_name, created_by_staff_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, NULLIF($2, '')::uuid, NOW(), NOW())
		RETURNING id::text, route_name, COALESCE(created_by_staff_id::text, ''), created_at, updated_at
	`, routeName, staffID,
	).Scan(&rt.ID, &rt.RouteName, &rt.CreatedByStaffID, &rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, outbound.ErrRouteNameExists
		}
		return nil, err
	}
	return &rt, nil
}

func (r *StaffRepository) DeleteRouteIfOwner(ctx context.Context, staffID, routeID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM routes WHERE id = $1::uuid AND created_by_staff_id = $2::uuid
	`, routeID, staffID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return outbound.ErrNotFound
	}
	return nil
}

func (r *StaffRepository) ListRoutes(ctx context.Context, in inbound.ListRoutesInput) ([]inbound.Route, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	q := strings.Builder{}
	q.WriteString(`
		SELECT id::text, route_name, COALESCE(created_by_staff_id::text, ''), created_at, updated_at
		FROM routes
		WHERE 1=1
	`)
	args := []any{}
	n := 1
	if strings.TrimSpace(in.CreatedByStaffID) != "" {
		q.WriteString(` AND created_by_staff_id = $` + strconv.Itoa(n) + `::uuid`)
		args = append(args, in.CreatedByStaffID)
		n++
	}
	q.WriteString(` ORDER BY route_name ASC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1))
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]inbound.Route, 0)
	for rows.Next() {
		var rt inbound.Route
		if err := rows.Scan(&rt.ID, &rt.RouteName, &rt.CreatedByStaffID, &rt.CreatedAt, &rt.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rt)
	}
	return out, rows.Err()
}
