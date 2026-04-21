package postgres

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/shared/s3media"
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
	// JOIN found_items to get current image_keys rather than the cached
	// image_urls stored in the notifications row, which may be expired
	// presigned URLs from old rows. The presigner in ListNotifications then
	// converts those keys to fresh signed URLs at read-time.
	q.WriteString(`
		SELECT
			n.id::text, n.passenger_id::text, n.lost_report_id::text, n.found_item_id::text,
			n.similarity_score, n.item_name,
			COALESCE(fi.image_keys, '{}'::text[]), COALESCE(fi.primary_image_key, ''),
			n.created_at, n.read_at
		FROM passenger_match_notifications n
		LEFT JOIN found_items fi ON fi.id = n.found_item_id
		WHERE n.passenger_id = $1::uuid
	`)
	args := []any{passengerID}
	n := 2
	if unreadOnly {
		q.WriteString(` AND n.read_at IS NULL`)
	}
	if !createdBefore.IsZero() {
		q.WriteString(` AND n.created_at < $` + strconv.Itoa(n))
		args = append(args, createdBefore)
		n++
	}
	q.WriteString(` ORDER BY n.created_at DESC LIMIT $` + strconv.Itoa(n))
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get a presigner once for all rows (nil if AWS env vars aren't set).
	presigner, presignErr := s3media.GetPresigner(ctx)
	if presignErr != nil {
		log.Printf("notifications: presigner unavailable (%v) — images will not load", presignErr)
	}

	out := make([]inbound.PassengerMatchNotification, 0)
	for rows.Next() {
		var nt inbound.PassengerMatchNotification
		var rawKeys pgtype.FlatArray[string]
		var rawPrimary string
		var readAt pgtype.Timestamptz
		if err := rows.Scan(
			&nt.ID, &nt.PassengerID, &nt.LostReportID, &nt.FoundItemID,
			&nt.SimilarityScore, &nt.ItemName,
			&rawKeys, &rawPrimary,
			&nt.CreatedAt, &readAt,
		); err != nil {
			return nil, err
		}
		if readAt.Valid {
			nt.ReadAt = readAt.Time
		}

		// Convert stored S3 keys to fresh presigned URLs. If presigning fails
		// (e.g. AWS not configured in dev) we silently skip that image so the
		// rest of the notification still renders.
		if presigner != nil {
			keys := []string(rawKeys)
			urls := make([]string, 0, len(keys))
			for _, k := range keys {
				if strings.TrimSpace(k) == "" {
					continue
				}
				u, err := presigner.PresignGet(ctx, k)
				if err == nil && strings.TrimSpace(u) != "" {
					urls = append(urls, u)
				}
			}
			nt.ImageURLs = urls

			if strings.TrimSpace(rawPrimary) != "" {
				if u, err := presigner.PresignGet(ctx, rawPrimary); err == nil {
					nt.PrimaryImageURL = u
				}
			}
			if nt.PrimaryImageURL == "" && len(urls) > 0 {
				nt.PrimaryImageURL = urls[0]
			}
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
