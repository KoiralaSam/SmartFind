package postgres

import (
	"context"
	"errors"
	"strings"
)

func (r *PassengerRepository) UpdateLostReportStatus(ctx context.Context, passengerID string, lostReportID string, status string) error {
	if r == nil || r.pool == nil {
		return errors.New("db pool is nil")
	}
	passengerID = strings.TrimSpace(passengerID)
	lostReportID = strings.TrimSpace(lostReportID)
	status = strings.TrimSpace(status)
	if passengerID == "" || lostReportID == "" {
		return errors.New("passengerID and lostReportID are required")
	}
	if status == "" {
		return errors.New("status is required")
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE lost_reports
		SET status = $1::lost_report_status, updated_at = NOW()
		WHERE id = $2::uuid AND reporter_passenger_id = $3::uuid
	`, status, lostReportID, passengerID)
	return err
}
