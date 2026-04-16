package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
)

func (r *PassengerRepository) ListNotifications(ctx context.Context, passengerID string, limit int, unreadOnly bool, createdBefore time.Time) ([]inbound.PassengerMatchNotification, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("db pool is nil")
	}
	passengerID = strings.TrimSpace(passengerID)
	if passengerID == "" {
		return nil, errors.New("passengerID is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	q := strings.Builder{}
	q.WriteString(`
		SELECT
			id::text, passenger_id::text, lost_report_id::text, found_item_id::text,
			similarity_score, item_name,
			COALESCE(image_urls, '{}'::text[]), COALESCE(primary_image_url, ''),
			created_at, read_at
		FROM passenger_match_notifications
		WHERE passenger_id = $1::uuid
	`)
	args := []any{passengerID}
	n := 2
	if unreadOnly {
		q.WriteString(` AND read_at IS NULL`)
	}
	if !createdBefore.IsZero() {
		q.WriteString(` AND created_at < $` + strconv.Itoa(n))
		args = append(args, createdBefore)
		n++
	}
	q.WriteString(` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(n))
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]inbound.PassengerMatchNotification, 0)
	for rows.Next() {
		var nt inbound.PassengerMatchNotification
		var imageURLs pgtype.FlatArray[string]
		var readAt pgtype.Timestamptz
		if err := rows.Scan(
			&nt.ID, &nt.PassengerID, &nt.LostReportID, &nt.FoundItemID,
			&nt.SimilarityScore, &nt.ItemName,
			&imageURLs, &nt.PrimaryImageURL,
			&nt.CreatedAt, &readAt,
		); err != nil {
			return nil, err
		}
		if imageURLs != nil {
			nt.ImageURLs = []string(imageURLs)
		}
		if readAt.Valid {
			nt.ReadAt = readAt.Time
		}
		out = append(out, nt)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return out, nil
}

func (r *PassengerRepository) MarkNotificationsRead(ctx context.Context, passengerID string, notificationIDs []string) error {
	if r == nil || r.pool == nil {
		return errors.New("db pool is nil")
	}
	passengerID = strings.TrimSpace(passengerID)
	if passengerID == "" {
		return errors.New("passengerID is required")
	}
	if len(notificationIDs) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(notificationIDs))
	args := make([]any, 0, 1+len(notificationIDs))
	args = append(args, passengerID)

	for i, id := range notificationIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		args = append(args, id)
		placeholders = append(placeholders, "$"+strconv.Itoa(i+2)+"::uuid")
	}
	if len(placeholders) == 0 {
		return nil
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE passenger_match_notifications
		SET read_at = COALESCE(read_at, NOW())
		WHERE passenger_id = $1::uuid
		  AND id = ANY(ARRAY[`+strings.Join(placeholders, ",")+`])
	`, args...)
	return err
}
