package postgres

import (
	"context"
	"time"
)

func (r *PassengerRepository) CountNotificationsForLostReportSince(ctx context.Context, lostReportID string, window time.Duration) (int, error) {
	return r.countNotificationsSince(ctx, "lost_report_id", lostReportID, window)
}

func (r *PassengerRepository) CountNotificationsForPassengerSince(ctx context.Context, passengerID string, window time.Duration) (int, error) {
	return r.countNotificationsSince(ctx, "passenger_id", passengerID, window)
}

func (r *PassengerRepository) countNotificationsSince(ctx context.Context, col string, id string, window time.Duration) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM passenger_match_notifications
		WHERE `+col+` = $1::uuid
		  AND created_at >= NOW() - ($2::int * interval '1 second')
	`, id, int(window.Seconds())).Scan(&n)
	return n, err
}

func (r *PassengerRepository) InsertMatchNotification(ctx context.Context, passengerID, lostReportID, foundItemID string, similarity float64, itemName string, imageKeys []string, primaryImageKey string) (bool, error) {
	if primaryImageKey == "" && len(imageKeys) > 0 {
		primaryImageKey = imageKeys[0]
	}

	tag, err := r.pool.Exec(ctx, `
		INSERT INTO passenger_match_notifications (
			passenger_id, lost_report_id, found_item_id,
			similarity_score, item_name, image_urls, primary_image_url
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid,
			$4, $5, $6::text[], $7
		)
		ON CONFLICT (lost_report_id, found_item_id) DO NOTHING
	`, passengerID, lostReportID, foundItemID, similarity, itemName, imageKeys, primaryImageKey)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PassengerRepository) UpdateLostReportMatchAudit(ctx context.Context, lostReportID string, checkedAt *time.Time, emailedAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE lost_reports
		SET
			match_last_checked_at = COALESCE($2, match_last_checked_at),
			match_last_emailed_at = COALESCE($3, match_last_emailed_at),
			updated_at = NOW()
		WHERE id = $1::uuid
	`, lostReportID, checkedAt, emailedAt)
	return err
}
