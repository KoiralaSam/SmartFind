package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	grpcadapter "smartfind/services/passenger-service/internal/adapters/secondary/grpc"
	"smartfind/shared/db"
	"smartfind/shared/env"
	"smartfind/shared/pgvector"
	staffpb "smartfind/shared/proto/staff"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type lostReportRow struct {
	LostReportID     string
	PassengerID      string
	PassengerEmail   string
	LostItemName     string
	EmbeddingLiteral string
	LastCheckedAt    pgtype.Timestamptz
	LastEmailedAt    pgtype.Timestamptz
}

type insertedMatch struct {
	FoundItemID     string
	ItemName        string
	SimilarityScore float64
	PrimaryImageURL string
	ImageURLs       []string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	dbURL := strings.TrimSpace(env.GetString("DATABASE_URL", ""))
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if err := db.InitDB(ctx, dbURL); err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	defer func() {
		if db.Pool != nil {
			db.Pool.Close()
		}
	}()

	staffClient, err := grpcadapter.NewStaffClient()
	if err != nil {
		log.Fatalf("failed to init staff grpc client: %v", err)
	}
	defer staffClient.Close()

	if err := runOnce(ctx, db.GetDB(), staffClient.Client); err != nil {
		log.Fatalf("match-worker failed: %v", err)
	}
}

func runOnce(ctx context.Context, pool *pgxpool.Pool, staffClient staffpb.StaffServiceClient) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	if staffClient == nil {
		return errors.New("staff service client is nil")
	}

	// ... body moved below by gofmt; kept as separate function signature change
	return runOnceWithPool(ctx, pool, staffClient)
}

func runOnceWithPool(ctx context.Context, pool *pgxpool.Pool, staffClient staffpb.StaffServiceClient) error {
	minSim := env.GetFloat("MATCH_MIN_SIMILARITY", 0.82)
	if minSim < 0 {
		minSim = 0
	}
	if minSim > 1 {
		minSim = 1
	}
	searchLimit := env.GetInt("MATCH_WORKER_SEARCH_LIMIT", 10)
	if searchLimit <= 0 {
		searchLimit = 10
	}
	if searchLimit > 50 {
		searchLimit = 50
	}

	maxPerReport := env.GetInt("MATCH_MAX_NOTIFS_PER_REPORT_PER_DAY", 3)
	if maxPerReport <= 0 {
		maxPerReport = 3
	}
	maxPerPassenger := env.GetInt("MATCH_MAX_NOTIFS_PER_PASSENGER_PER_DAY", 10)
	if maxPerPassenger <= 0 {
		maxPerPassenger = 10
	}

	emailCooldownMin := env.GetInt("MATCH_EMAIL_COOLDOWN_MINUTES", 60)
	if emailCooldownMin <= 0 {
		emailCooldownMin = 60
	}
	emailMaxItems := env.GetInt("MATCH_EMAIL_MAX_ITEMS", 5)
	if emailMaxItems <= 0 {
		emailMaxItems = 5
	}
	if emailMaxItems > 20 {
		emailMaxItems = 20
	}

	batch := env.GetInt("MATCH_WORKER_BATCH_LIMIT", 200)
	if batch <= 0 {
		batch = 200
	}

	rows, err := pool.Query(ctx, `
		SELECT
			lr.id::text,
			lr.reporter_passenger_id::text,
			p.email,
			lr.item_name,
			COALESCE(lre.embedding::text, ''),
			lr.match_last_checked_at,
			lr.match_last_emailed_at
		FROM lost_reports lr
		JOIN passengers p ON p.id = lr.reporter_passenger_id
		LEFT JOIN lost_report_embeddings lre ON lre.lost_report_id = lr.id
		WHERE lr.status = 'open'
		ORDER BY lr.created_at DESC
		LIMIT $1
	`, batch)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var lr lostReportRow
		if err := rows.Scan(
			&lr.LostReportID,
			&lr.PassengerID,
			&lr.PassengerEmail,
			&lr.LostItemName,
			&lr.EmbeddingLiteral,
			&lr.LastCheckedAt,
			&lr.LastEmailedAt,
		); err != nil {
			return err
		}
		if strings.TrimSpace(lr.EmbeddingLiteral) == "" {
			continue
		}

		embedding, err := pgvector.ParseLiteral(lr.EmbeddingLiteral)
		if err != nil || len(embedding) != 1536 {
			continue
		}

		mctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		resp, err := staffClient.SearchFoundItemMatchesByEmbedding(mctx, &staffpb.SearchFoundItemMatchesByEmbeddingRequest{
			QueryEmbedding: embedding,
			Limit:          int32(searchLimit),
			MinSimilarity:  minSim,
		})
		cancel()
		if err != nil {
			log.Printf("staff search failed lost_report_id=%s: %v", lr.LostReportID, err)
			_ = updateLastChecked(ctx, pool, lr.LostReportID, now)
			continue
		}

		reportCount, err := countNotificationsSince(ctx, pool, "lost_report_id", lr.LostReportID, 24*time.Hour)
		if err != nil {
			return err
		}
		passengerCount, err := countNotificationsSince(ctx, pool, "passenger_id", lr.PassengerID, 24*time.Hour)
		if err != nil {
			return err
		}
		remaining := minInt(maxPerReport-reportCount, maxPerPassenger-passengerCount)
		if remaining <= 0 {
			_ = updateLastChecked(ctx, pool, lr.LostReportID, now)
			continue
		}

		inserted := make([]insertedMatch, 0)
		for _, m := range resp.GetMatches() {
			if remaining <= 0 {
				break
			}
			if m == nil || m.GetItem() == nil {
				continue
			}
			it := m.GetItem()
			ok, err := insertNotification(ctx, pool, lr.PassengerID, lr.LostReportID, it.GetId(), m.GetSimilarityScore(), it.GetItemName(), it.GetImageUrls(), it.GetPrimaryImageUrl())
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			inserted = append(inserted, insertedMatch{
				FoundItemID:     it.GetId(),
				ItemName:        it.GetItemName(),
				SimilarityScore: m.GetSimilarityScore(),
				PrimaryImageURL: it.GetPrimaryImageUrl(),
				ImageURLs:       it.GetImageUrls(),
			})
			remaining--
		}

		_ = updateLastChecked(ctx, pool, lr.LostReportID, now)

		if len(inserted) == 0 {
			continue
		}

		if lr.LastEmailedAt.Valid && lr.LastEmailedAt.Time.After(now.Add(-time.Duration(emailCooldownMin)*time.Minute)) {
			continue
		}

		if err := sendSendgridEmail(ctx, lr.PassengerEmail, lr.LostReportID, lr.LostItemName, inserted, emailMaxItems); err != nil {
			log.Printf("sendgrid email failed passenger_id=%s lost_report_id=%s: %v", lr.PassengerID, lr.LostReportID, err)
			continue
		}
		_ = updateLastEmailed(ctx, pool, lr.LostReportID, now)
	}
	return rows.Err()
}

func countNotificationsSince(ctx context.Context, pool *pgxpool.Pool, col string, id string, window time.Duration) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM passenger_match_notifications
		WHERE `+col+` = $1::uuid
		  AND created_at >= NOW() - ($2::int * interval '1 second')
	`, id, int(window.Seconds())).Scan(&n)
	return n, err
}

func insertNotification(ctx context.Context, pool *pgxpool.Pool, passengerID, lostReportID, foundItemID string, similarity float64, itemName string, imageURLs []string, primaryImageURL string) (bool, error) {
	if strings.TrimSpace(primaryImageURL) == "" && len(imageURLs) > 0 {
		primaryImageURL = imageURLs[0]
	}
	if primaryImageURL == "" {
		primaryImageURL = ""
	}

	tag, err := pool.Exec(ctx, `
		INSERT INTO passenger_match_notifications (
			passenger_id, lost_report_id, found_item_id,
			similarity_score, item_name, image_urls, primary_image_url
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid,
			$4, $5, $6::text[], $7
		)
		ON CONFLICT (lost_report_id, found_item_id) DO NOTHING
	`, passengerID, lostReportID, foundItemID, similarity, itemName, imageURLs, primaryImageURL)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func updateLastChecked(ctx context.Context, pool *pgxpool.Pool, lostReportID string, t time.Time) error {
	_, err := pool.Exec(ctx, `
		UPDATE lost_reports
		SET match_last_checked_at = $2, updated_at = NOW()
		WHERE id = $1::uuid
	`, lostReportID, t)
	return err
}

func updateLastEmailed(ctx context.Context, pool *pgxpool.Pool, lostReportID string, t time.Time) error {
	_, err := pool.Exec(ctx, `
		UPDATE lost_reports
		SET match_last_emailed_at = $2, updated_at = NOW()
		WHERE id = $1::uuid
	`, lostReportID, t)
	return err
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sendSendgridEmail(ctx context.Context, toEmail string, lostReportID string, lostItemName string, matches []insertedMatch, maxItems int) error {
	apiKey := strings.TrimSpace(os.Getenv("SENDGRID_API_KEY"))
	fromEmail := strings.TrimSpace(os.Getenv("SENDGRID_FROM_EMAIL"))
	publicBase := strings.TrimSpace(os.Getenv("PUBLIC_APP_BASE_URL"))
	if apiKey == "" || strings.EqualFold(apiKey, "REPLACE_ME") || fromEmail == "" || strings.EqualFold(fromEmail, "REPLACE_ME") {
		return errors.New("SENDGRID_API_KEY and SENDGRID_FROM_EMAIL are required")
	}
	if strings.TrimSpace(toEmail) == "" {
		return errors.New("to email is empty")
	}

	if maxItems <= 0 {
		maxItems = 5
	}
	if len(matches) > maxItems {
		matches = matches[:maxItems]
	}

	link := strings.TrimRight(publicBase, "/") + "/passenger/chat"
	if strings.TrimSpace(publicBase) == "" {
		link = "/passenger/chat"
	}

	subject := "We found potential matches for your lost item"
	if strings.TrimSpace(lostItemName) != "" {
		subject = fmt.Sprintf("Possible matches for your lost %s", lostItemName)
	}

	text := buildTextEmail(lostReportID, link, matches)
	html := buildHTMLEmail(lostReportID, link, matches)

	payload := map[string]any{
		"personalizations": []any{
			map[string]any{
				"to":      []any{map[string]any{"email": toEmail}},
				"subject": subject,
			},
		},
		"from": map[string]any{"email": fromEmail},
		"content": []any{
			map[string]any{"type": "text/plain", "value": text},
			map[string]any{"type": "text/html", "value": html},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("sendgrid status %d", res.StatusCode)
	}
	return nil
}

func buildTextEmail(lostReportID string, link string, matches []insertedMatch) string {
	var b strings.Builder
	b.WriteString("We found potential matches for your lost item.\n\n")
	b.WriteString("Lost report ID: " + lostReportID + "\n\n")
	for i, m := range matches {
		b.WriteString(fmt.Sprintf("%d) %s (score: %.2f)\n", i+1, safe(m.ItemName, "Item"), m.SimilarityScore))
		if strings.TrimSpace(m.PrimaryImageURL) != "" {
			b.WriteString("   Photo: " + m.PrimaryImageURL + "\n")
		}
	}
	b.WriteString("\nOpen the app to review and file a claim:\n" + link + "\n")
	return b.String()
}

func buildHTMLEmail(lostReportID string, link string, matches []insertedMatch) string {
	var b strings.Builder
	b.WriteString("<div>")
	b.WriteString("<p>We found potential matches for your lost item.</p>")
	b.WriteString("<p><strong>Lost report ID:</strong> " + htmlEscape(lostReportID) + "</p>")
	b.WriteString("<ul>")
	for _, m := range matches {
		b.WriteString("<li>")
		b.WriteString("<div><strong>" + htmlEscape(safe(m.ItemName, "Item")) + "</strong> (score: " + fmt.Sprintf("%.2f", m.SimilarityScore) + ")</div>")
		if strings.TrimSpace(m.PrimaryImageURL) != "" {
			b.WriteString("<div><img src=\"" + htmlEscape(m.PrimaryImageURL) + "\" alt=\"match\" style=\"max-width:240px;border-radius:8px;\" /></div>")
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	b.WriteString("<p><a href=\"" + htmlEscape(link) + "\">Open SmartFind to review and file a claim</a></p>")
	b.WriteString("</div>")
	return b.String()
}

func safe(s string, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}
