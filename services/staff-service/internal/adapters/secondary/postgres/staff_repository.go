package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"smartfind/services/staff-service/internal/core/domain"
	"smartfind/services/staff-service/internal/core/ports/outbound"
)

type StaffRepository struct {
	pool *pgxpool.Pool
}

func NewStaffRepository(pool *pgxpool.Pool) outbound.StaffRepository {
	return &StaffRepository{pool: pool}
}

func (r *StaffRepository) GetByEmail(ctx context.Context, email string) (*domain.Staff, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, full_name, email, COALESCE(password_hash, ''), created_at, updated_at
		FROM staff
		WHERE LOWER(TRIM(email)) = LOWER(TRIM($1))
	`, email)

	var s domain.Staff
	err := row.Scan(&s.ID, &s.FullName, &s.Email, &s.PasswordHash, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StaffRepository) Create(ctx context.Context, staff domain.Staff) (*domain.Staff, error) {
	if staff.CreatedAt.IsZero() {
		staff.CreatedAt = time.Now()
	}
	if staff.UpdatedAt.IsZero() {
		staff.UpdatedAt = staff.CreatedAt
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO staff (full_name, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, full_name, email, COALESCE(password_hash, ''), created_at, updated_at
	`, staff.FullName, staff.Email, nullIfEmpty(staff.PasswordHash), staff.CreatedAt, staff.UpdatedAt,
	).Scan(&staff.ID, &staff.FullName, &staff.Email, &staff.PasswordHash, &staff.CreatedAt, &staff.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, outbound.ErrStaffEmailExists
		}
		return nil, err
	}
	return &staff, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
