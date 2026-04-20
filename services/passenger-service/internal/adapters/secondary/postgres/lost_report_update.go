package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/services/passenger-service/internal/core/ports/outbound"
)

// UpdateLostReport applies the non-nil fields of `in` to the matching
// lost_reports row and returns the row after the update. If the row isn't
// owned by `in.PassengerID` or doesn't exist, outbound.ErrLostReportNotFound
// is returned. When `in` carries no updatable field the row is returned
// unchanged.
func (r *PassengerRepository) UpdateLostReport(ctx context.Context, in inbound.UpdateLostReportInput) (*inbound.LostReport, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("db pool is nil")
	}
	passengerID := strings.TrimSpace(in.PassengerID)
	lostReportID := strings.TrimSpace(in.LostReportID)
	if passengerID == "" || lostReportID == "" {
		return nil, errors.New("passengerID and lostReportID are required")
	}

	// Build the SET clause dynamically so callers can send partial patches
	// without wiping fields they didn't mention.
	sets := make([]string, 0, 14)
	args := make([]any, 0, 16)
	idx := 1
	addStr := func(column string, ptr *string) {
		if ptr == nil {
			return
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", column, idx))
		args = append(args, strings.TrimSpace(*ptr))
		idx++
	}
	addStr("item_name", in.ItemName)
	addStr("item_description", in.ItemDescription)
	addStr("item_type", in.ItemType)
	addStr("brand", in.Brand)
	addStr("model", in.Model)
	addStr("color", in.Color)
	addStr("material", in.Material)
	addStr("item_condition", in.ItemCondition)
	addStr("category", in.Category)
	addStr("location_lost", in.LocationLost)
	addStr("route_or_station", in.RouteOrStation)
	if in.RouteID != nil {
		sets = append(sets, fmt.Sprintf("route_id = NULLIF($%d, '')::uuid", idx))
		args = append(args, strings.TrimSpace(*in.RouteID))
		idx++
	}
	if in.DateLost != nil {
		sets = append(sets, fmt.Sprintf("date_lost = $%d", idx))
		args = append(args, *in.DateLost)
		idx++
	}

	// Always bump updated_at if anything actually changes.
	if len(sets) == 0 {
		rpt, err := r.GetLostReportForPassenger(ctx, passengerID, lostReportID)
		if err != nil {
			return nil, err
		}
		if rpt == nil {
			return nil, outbound.ErrLostReportNotFound
		}
		return rpt, nil
	}
	sets = append(sets, "updated_at = NOW()")

	// Final positional args: id + passenger_id for the WHERE clause.
	args = append(args, lostReportID, passengerID)
	whereIDIdx := idx
	wherePassengerIdx := idx + 1

	query := fmt.Sprintf(`
		UPDATE lost_reports
		SET %s
		WHERE id = $%d::uuid AND reporter_passenger_id = $%d::uuid
		RETURNING
			id::text, reporter_passenger_id::text,
			item_name, item_description, item_type, brand, model, color, material, item_condition,
			category, location_lost, route_or_station, COALESCE(route_id::text, ''), date_lost,
			status::text, created_at, updated_at
	`, strings.Join(sets, ", "), whereIDIdx, wherePassengerIdx)

	var rpt inbound.LostReport
	var routeID string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&rpt.ID, &rpt.ReporterPassengerID,
		&rpt.ItemName, &rpt.ItemDescription, &rpt.ItemType, &rpt.Brand, &rpt.Model, &rpt.Color, &rpt.Material, &rpt.ItemCondition,
		&rpt.Category, &rpt.LocationLost, &rpt.RouteOrStation, &routeID, &rpt.DateLost,
		&rpt.Status, &rpt.CreatedAt, &rpt.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, outbound.ErrLostReportNotFound
	}
	if err != nil {
		return nil, err
	}
	rpt.RouteID = routeID
	return &rpt, nil
}
