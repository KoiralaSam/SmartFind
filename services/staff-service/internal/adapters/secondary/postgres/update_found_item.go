package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"smartfind/services/staff-service/internal/core/ports/inbound"
	"smartfind/services/staff-service/internal/core/ports/outbound"
)

// UpdateFoundItem applies a partial update to a found item owned by the
// given staff member. Only fields present in the input (non-zero for scalar
// types, non-nil for pointer types) are included in the SET clause.
// The image key list is replaced entirely when in.ImageKeys is non-nil.
func (r *StaffRepository) UpdateFoundItem(ctx context.Context, in inbound.UpdateFoundItemInput) (*inbound.FoundItem, error) {
	if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.FoundItemID) == "" {
		return nil, errors.New("staff_id and found_item_id are required")
	}

	type col struct {
		expr string
		arg  any
	}
	var sets []col
	n := 1

	add := func(expr string, arg any) {
		sets = append(sets, col{expr: expr, arg: arg})
		n++
	}

	if v := strings.TrimSpace(in.ItemName); v != "" {
		add("item_name = $"+strconv.Itoa(n), v)
	}
	if v := in.ItemDescription; v != "" {
		add("item_description = $"+strconv.Itoa(n), strings.TrimSpace(v))
	}
	if v := strings.TrimSpace(in.ItemType); v != "" {
		add("item_type = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.Brand); v != "" {
		add("brand = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.Model); v != "" {
		add("model = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.Color); v != "" {
		add("color = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.Material); v != "" {
		add("material = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.ItemCondition); v != "" {
		add("item_condition = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.Category); v != "" {
		add("category = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.LocationFound); v != "" {
		add("location_found = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.RouteOrStation); v != "" {
		add("route_or_station = $"+strconv.Itoa(n), v)
	}
	if v := strings.TrimSpace(in.RouteID); v != "" {
		add("route_id = NULLIF($"+strconv.Itoa(n)+", '')::uuid", v)
	}
	if !in.DateFound.IsZero() {
		add("date_found = $"+strconv.Itoa(n), in.DateFound)
	}
	if in.ImageKeys != nil {
		add("image_keys = $"+strconv.Itoa(n)+"::text[]", pgtype.FlatArray[string](*in.ImageKeys))
	}
	if in.PrimaryImageKey != nil {
		add("primary_image_key = NULLIF($"+strconv.Itoa(n)+", '')", *in.PrimaryImageKey)
	}

	if len(sets) == 0 {
		return nil, errors.New("no fields to update")
	}

	// always bump updated_at
	sets = append(sets, col{expr: "updated_at = $" + strconv.Itoa(n), arg: time.Now()})
	n++

	var q strings.Builder
	q.WriteString("UPDATE found_items SET ")
	args := []any{}
	for i, s := range sets {
		if i > 0 {
			q.WriteString(", ")
		}
		q.WriteString(s.expr)
		args = append(args, s.arg)
	}
	// $n and $n+1 are staff_id and found_item_id for the WHERE clause
	q.WriteString(" WHERE id = $" + strconv.Itoa(n) + "::uuid AND posted_by_staff_id = $" + strconv.Itoa(n+1) + "::uuid")
	args = append(args, in.FoundItemID, in.StaffID)

	q.WriteString(`
		RETURNING
			id::text, posted_by_staff_id::text,
			item_name, COALESCE(item_description, ''), item_type, brand, model, color, material, item_condition,
			COALESCE(category, ''), COALESCE(location_found, ''), COALESCE(route_or_station, ''),
			COALESCE(route_id::text, ''), date_found,
			COALESCE(image_keys, '{}'::text[]), COALESCE(primary_image_key, ''),
			status::text, created_at, updated_at
	`)

	var it inbound.FoundItem
	var df pgtype.Date
	var imageKeys pgtype.FlatArray[string]
	err := r.pool.QueryRow(ctx, q.String(), args...).Scan(
		&it.ID, &it.PostedByStaffID,
		&it.ItemName, &it.ItemDescription, &it.ItemType, &it.Brand, &it.Model, &it.Color, &it.Material, &it.ItemCondition,
		&it.Category, &it.LocationFound, &it.RouteOrStation, &it.RouteID,
		&df, &imageKeys, &it.PrimaryImageKey, &it.Status, &it.CreatedAt, &it.UpdatedAt,
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
	if imageKeys != nil {
		it.ImageKeys = []string(imageKeys)
	}
	return &it, nil
}
