package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"smartfind/services/passenger-service/internal/core/domain"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/services/passenger-service/internal/core/ports/outbound"
)

type PassengerRepository struct {
	pool *pgxpool.Pool
}

func NewPassengerRepository(pool *pgxpool.Pool) outbound.PassengerRepository {
	return &PassengerRepository{pool: pool}
}

func (r *PassengerRepository) GetByID(ctx context.Context, id string) (*domain.Passenger, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, full_name, phone, avatar_url, created_at, updated_at
		FROM passengers
		WHERE id = $1
	`, id)

	var p domain.Passenger
	err := row.Scan(&p.ID, &p.Email, &p.FullName, &p.Phone, &p.AvatarURL, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PassengerRepository) GetByEmail(ctx context.Context, email string) (*domain.Passenger, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, full_name, phone, avatar_url, created_at, updated_at
		FROM passengers
		WHERE email = $1
	`, email)

	var p domain.Passenger
	err := row.Scan(&p.ID, &p.Email, &p.FullName, &p.Phone, &p.AvatarURL, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PassengerRepository) Create(ctx context.Context, passenger domain.Passenger) (*domain.Passenger, error) {
	if passenger.CreatedAt.IsZero() {
		passenger.CreatedAt = time.Now()
	}
	if passenger.UpdatedAt.IsZero() {
		passenger.UpdatedAt = passenger.CreatedAt
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO passengers (email, full_name, phone, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, email, full_name, phone, avatar_url, created_at, updated_at
	`, passenger.Email, passenger.FullName, passenger.Phone, passenger.AvatarURL, passenger.CreatedAt, passenger.UpdatedAt,
	).Scan(&passenger.ID, &passenger.Email, &passenger.FullName, &passenger.Phone, &passenger.AvatarURL, &passenger.CreatedAt, &passenger.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &passenger, nil
}

func (r *PassengerRepository) Update(ctx context.Context, passenger domain.Passenger) error {
	passenger.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE passengers
		SET email = $2, full_name = $3, phone = $4, avatar_url = $5, updated_at = $6
		WHERE id = $1
	`, passenger.ID, passenger.Email, passenger.FullName, passenger.Phone, passenger.AvatarURL, passenger.UpdatedAt)
	return err
}

func (r *PassengerRepository) CreateLostReport(ctx context.Context, report inbound.LostReport) (*inbound.LostReport, error) {
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now()
	}
	if report.UpdatedAt.IsZero() {
		report.UpdatedAt = report.CreatedAt
	}
	if report.Status == "" {
		report.Status = "open"
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO lost_reports (
			reporter_passenger_id,
			item_name, item_description, item_type, brand, model, color, material, item_condition,
			category, location_lost, route_or_station, route_id, date_lost,
			status, created_at, updated_at
		) VALUES (
			$1,
			$2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, NULLIF($13, '')::uuid, $14,
			$15, $16, $17
		)
		RETURNING id::text, created_at, updated_at
	`, report.ReporterPassengerID,
		report.ItemName, report.ItemDescription, report.ItemType, report.Brand, report.Model, report.Color, report.Material, report.ItemCondition,
		report.Category, report.LocationLost, report.RouteOrStation, report.RouteID, report.DateLost,
		report.Status, report.CreatedAt, report.UpdatedAt,
	).Scan(&report.ID, &report.CreatedAt, &report.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *PassengerRepository) ListLostReports(ctx context.Context, passengerID string, status string) ([]inbound.LostReport, error) {
	var rows pgx.Rows
	var err error

	if status == "" {
		rows, err = r.pool.Query(ctx, `
			SELECT
				id, reporter_passenger_id,
				item_name, item_description, item_type, brand, model, color, material, item_condition,
				category, location_lost, route_or_station, COALESCE(route_id::text, ''), date_lost,
				status, created_at, updated_at
			FROM lost_reports
			WHERE reporter_passenger_id = $1
			ORDER BY created_at DESC
		`, passengerID)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT
				id, reporter_passenger_id,
				item_name, item_description, item_type, brand, model, color, material, item_condition,
				category, location_lost, route_or_station, COALESCE(route_id::text, ''), date_lost,
				status, created_at, updated_at
			FROM lost_reports
			WHERE reporter_passenger_id = $1 AND status = $2
			ORDER BY created_at DESC
		`, passengerID, status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]inbound.LostReport, 0)
	for rows.Next() {
		var rpt inbound.LostReport
		var routeID string
		if scanErr := rows.Scan(
			&rpt.ID, &rpt.ReporterPassengerID,
			&rpt.ItemName, &rpt.ItemDescription, &rpt.ItemType, &rpt.Brand, &rpt.Model, &rpt.Color, &rpt.Material, &rpt.ItemCondition,
			&rpt.Category, &rpt.LocationLost, &rpt.RouteOrStation, &routeID, &rpt.DateLost,
			&rpt.Status, &rpt.CreatedAt, &rpt.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		rpt.RouteID = routeID
		out = append(out, rpt)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func (r *PassengerRepository) DeleteLostReport(ctx context.Context, passengerID string, lostReportID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM lost_reports
		WHERE id = $1 AND reporter_passenger_id = $2
	`, lostReportID, passengerID)
	return err
}

func (r *PassengerRepository) CreateItemClaim(ctx context.Context, claim inbound.ItemClaim) (*inbound.ItemClaim, error) {
	if claim.CreatedAt.IsZero() {
		claim.CreatedAt = time.Now()
	}
	if claim.UpdatedAt.IsZero() {
		claim.UpdatedAt = claim.CreatedAt
	}
	if claim.Status == "" {
		claim.Status = "pending"
	}

	err := r.pool.QueryRow(ctx, `
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
	return &claim, nil
}
