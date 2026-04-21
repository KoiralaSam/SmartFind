package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
)

func (r *PassengerRepository) GetLostReportForPassenger(ctx context.Context, passengerID string, lostReportID string) (*inbound.LostReport, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("db pool is nil")
	}
	passengerID = strings.TrimSpace(passengerID)
	lostReportID = strings.TrimSpace(lostReportID)
	if passengerID == "" || lostReportID == "" {
		return nil, errors.New("passengerID and lostReportID are required")
	}

	var rpt inbound.LostReport
	var routeID string
	var lastChecked *time.Time
	var lastEmailed *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT
			id, reporter_passenger_id,
			item_name, item_description, item_type, brand, model, color, material, item_condition,
			category, location_lost, route_or_station, COALESCE(route_id::text, ''), date_lost,
			status, created_at, updated_at, match_last_checked_at, match_last_emailed_at
		FROM lost_reports
		WHERE id = $1::uuid AND reporter_passenger_id = $2::uuid
	`, lostReportID, passengerID).Scan(
		&rpt.ID, &rpt.ReporterPassengerID,
		&rpt.ItemName, &rpt.ItemDescription, &rpt.ItemType, &rpt.Brand, &rpt.Model, &rpt.Color, &rpt.Material, &rpt.ItemCondition,
		&rpt.Category, &rpt.LocationLost, &rpt.RouteOrStation, &routeID, &rpt.DateLost,
		&rpt.Status, &rpt.CreatedAt, &rpt.UpdatedAt, &lastChecked, &lastEmailed,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rpt.RouteID = routeID
	rpt.LastMatchCheckedAt = lastChecked
	rpt.LastMatchEmailedAt = lastEmailed
	return &rpt, nil
}
