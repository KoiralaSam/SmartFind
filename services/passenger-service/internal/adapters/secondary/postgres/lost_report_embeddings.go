package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

func (r *PassengerRepository) UpsertLostReportEmbedding(ctx context.Context, lostReportID string, embedding []float32) error {
	if strings.TrimSpace(lostReportID) == "" {
		return errors.New("lostReportID is required")
	}
	if len(embedding) != 1536 {
		return errors.New("embedding must have 1536 dimensions")
	}
	vecLit := vectorLiteral(embedding)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO lost_report_embeddings (lost_report_id, embedding)
		VALUES (NULLIF($1, '')::uuid, $2::vector)
		ON CONFLICT (lost_report_id) DO UPDATE
		SET embedding = EXCLUDED.embedding
	`, lostReportID, vecLit)
	return err
}

func vectorLiteral(vec []float32) string {
	var b strings.Builder
	b.Grow(len(vec) * 8)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
