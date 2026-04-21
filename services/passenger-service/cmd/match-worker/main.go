package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	grpcadapter "smartfind/services/passenger-service/internal/adapters/secondary/grpc"
	"smartfind/shared/db"
	"smartfind/shared/env"
	"smartfind/shared/pgvector"
	"smartfind/shared/s3media"
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
	FoundItemID      string
	ItemName         string
	SimilarityScore  float64
	PrimaryImageKey  string
	ImageKeys        []string
}

// emailMatch is the same row plus presigned image URLs for HTML email clients.
type emailMatch struct {
	insertedMatch
	ImageURLs []string
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
	minSim := env.GetFloat("MATCH_MIN_SIMILARITY", 0.65)
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
			// Store raw S3 keys (not presigned URLs) so the passenger-service
			// can generate fresh presigned URLs at read-time. Presigned URLs
			// expire in ~10 minutes; keys never expire.
			ok, err := insertNotification(ctx, pool, lr.PassengerID, lr.LostReportID, it.GetId(), m.GetSimilarityScore(), it.GetItemName(), it.GetImageKeys(), it.GetPrimaryImageKey())
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
				PrimaryImageKey: it.GetPrimaryImageKey(),
				ImageKeys:       it.GetImageKeys(),
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

		if err := sendMatchEmail(ctx, lr.PassengerEmail, lr.LostReportID, lr.LostItemName, inserted, emailMaxItems); err != nil {
			// sendMatchEmail presigns S3 keys for inline images; if Mailtrap or S3
			// is misconfigured, we still keep DB notifications — only log the error.
			log.Printf("match email failed passenger_id=%s lost_report_id=%s: %v", lr.PassengerID, lr.LostReportID, err)
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

// insertNotification stores raw S3 keys (not presigned URLs) in the
// image_urls / primary_image_url columns. Presigning happens at read-time
// inside passenger-service ListNotifications so images never show as broken.
func insertNotification(ctx context.Context, pool *pgxpool.Pool, passengerID, lostReportID, foundItemID string, similarity float64, itemName string, imageKeys []string, primaryImageKey string) (bool, error) {
	if strings.TrimSpace(primaryImageKey) == "" && len(imageKeys) > 0 {
		primaryImageKey = imageKeys[0]
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
	`, passengerID, lostReportID, foundItemID, similarity, itemName, imageKeys, primaryImageKey)
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

// sendMatchEmail delivers the match digest via Mailtrap's Send API
// (https://send.api.mailtrap.io/api/send). The transport was migrated off
// SendGrid; the call site falls back gracefully when the Mailtrap token is
// unset so the cron still inserts notifications into the database and simply
// skips the email send.
//
// Required env:
//
//	MAILTRAP_API_TOKEN      - API token with "Send Email" scope
//	MAILTRAP_FROM_EMAIL     - verified sender address for your Mailtrap domain
//
// Optional env:
//
//	MAILTRAP_FROM_NAME      - human-readable sender name (defaults to "SmartFind")
//	MAILTRAP_API_URL        - override the endpoint (e.g. sandbox inbox URL)
//	PUBLIC_APP_BASE_URL     - base URL used to build deep links in the email
func sendMatchEmail(ctx context.Context, toEmail string, lostReportID string, lostItemName string, matches []insertedMatch, maxItems int) error {
	apiToken := strings.TrimSpace(os.Getenv("MAILTRAP_API_TOKEN"))
	fromEmail := strings.TrimSpace(os.Getenv("MAILTRAP_FROM_EMAIL"))
	fromName := strings.TrimSpace(os.Getenv("MAILTRAP_FROM_NAME"))
	apiURL := strings.TrimSpace(os.Getenv("MAILTRAP_API_URL"))
	publicBase := strings.TrimSpace(os.Getenv("PUBLIC_APP_BASE_URL"))

	if fromName == "" {
		fromName = "SmartFind"
	}
	if apiURL == "" {
		apiURL = "https://send.api.mailtrap.io/api/send"
	}
	if apiToken == "" || strings.EqualFold(apiToken, "REPLACE_ME") ||
		fromEmail == "" || strings.EqualFold(fromEmail, "REPLACE_ME") {
		return errors.New("MAILTRAP_API_TOKEN and MAILTRAP_FROM_EMAIL are required")
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
	if publicBase == "" {
		link = "/passenger/chat"
	}

	subject := "We found potential matches for your lost item"
	if strings.TrimSpace(lostItemName) != "" {
		subject = fmt.Sprintf("Possible matches for your lost %s", lostItemName)
	}

	emailMatches := enrichMatchesForEmail(ctx, matches)
	text := buildTextEmail(lostReportID, link, lostItemName, toEmail, emailMatches)
	htmlBody := buildHTMLEmail(lostReportID, link, lostItemName, toEmail, emailMatches)

	payload := map[string]any{
		"from": map[string]any{
			"email": fromEmail,
			"name":  fromName,
		},
		"to":       []any{map[string]any{"email": toEmail}},
		"subject":  subject,
		"text":     text,
		"html":     htmlBody,
		"category": "smartfind-match-notification",
		"custom_variables": map[string]any{
			"lost_report_id": lostReportID,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Api-Token", apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		buf := make([]byte, 512)
		n, _ := res.Body.Read(buf)
		return fmt.Errorf("mailtrap status %d: %s", res.StatusCode, strings.TrimSpace(string(buf[:n])))
	}
	return nil
}

func orderedUniqueImageKeys(primary string, keys []string, max int) []string {
	if max <= 0 {
		max = 5
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, max)
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	add(primary)
	for _, k := range keys {
		if len(out) >= max {
			break
		}
		add(k)
	}
	return out
}

func greetingFromEmail(addr string) string {
	addr = strings.TrimSpace(addr)
	at := strings.IndexByte(addr, '@')
	local := addr
	if at > 0 {
		local = strings.ToLower(strings.TrimSpace(addr[:at]))
	}
	if local == "" {
		return "there"
	}
	seps := func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == '+'
	}
	fields := strings.FieldsFunc(local, seps)
	if len(fields) == 0 || fields[0] == "" {
		return "there"
	}
	f := fields[0]
	if len(f) == 1 {
		return strings.ToUpper(f)
	}
	return strings.ToUpper(f[:1]) + f[1:]
}

func enrichMatchesForEmail(ctx context.Context, matches []insertedMatch) []emailMatch {
	p, err := s3media.GetPresigner(ctx)
	if err != nil || p == nil {
		log.Printf("match email: presigner unavailable, sending without inline images: %v", err)
		out := make([]emailMatch, 0, len(matches))
		for _, m := range matches {
			out = append(out, emailMatch{insertedMatch: m})
		}
		return out
	}

	ttlSec := env.GetInt("MATCH_EMAIL_IMAGE_TTL_SECONDS", 7*24*3600)
	if ttlSec <= 0 {
		ttlSec = 7 * 24 * 3600
	}
	const maxPresignSeconds = 7 * 24 * 3600
	if ttlSec > maxPresignSeconds {
		ttlSec = maxPresignSeconds
	}
	ttl := time.Duration(ttlSec) * time.Second

	out := make([]emailMatch, 0, len(matches))
	for _, m := range matches {
		keys := orderedUniqueImageKeys(m.PrimaryImageKey, m.ImageKeys, 5)
		urls := make([]string, 0, len(keys))
		for _, k := range keys {
			u, err := p.PresignGetWithTTL(ctx, k, ttl)
			if err != nil || strings.TrimSpace(u) == "" {
				log.Printf("match email: presign failed for key=%s: %v", k, err)
				continue
			}
			urls = append(urls, u)
		}
		out = append(out, emailMatch{insertedMatch: m, ImageURLs: urls})
	}
	return out
}

func buildTextEmail(lostReportID, link, lostItemName, toEmail string, matches []emailMatch) string {
	var b strings.Builder
	b.WriteString("SmartFind — we found potential matches for your lost item.\n\n")
	b.WriteString(fmt.Sprintf("Hi %s,\n\n", greetingFromEmail(toEmail)))
	if strings.TrimSpace(lostItemName) != "" {
		b.WriteString(fmt.Sprintf("Regarding your lost item: %s\n\n", lostItemName))
	}
	b.WriteString("Lost report ID: " + lostReportID + "\n\n")
	for i, m := range matches {
		b.WriteString(fmt.Sprintf("%d) %s (similarity: %.0f%%)\n",
			i+1, safe(m.ItemName, "Item"), m.SimilarityScore*100))
		if len(m.ImageURLs) > 0 {
			b.WriteString("   Photo links (time-limited):\n")
			for _, u := range m.ImageURLs {
				b.WriteString("   - " + u + "\n")
			}
		}
	}
	b.WriteString("\nReview matches and file a claim in the app:\n" + link + "\n")
	return b.String()
}

func buildHTMLEmail(lostReportID, link, lostItemName, toEmail string, matches []emailMatch) string {
	greet := html.EscapeString(greetingFromEmail(toEmail))
	itemLabel := strings.TrimSpace(lostItemName)
	if itemLabel == "" {
		itemLabel = "your lost item"
	}
	itemEsc := html.EscapeString(itemLabel)

	moss := "#4a7c59"
	accent := "#e85d04"
	panel := "#e8f5e9"
	pageBG := "#f4f4f4"

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">`)
	b.WriteString(`<title>SmartFind — possible matches</title></head>`)
	b.WriteString(fmt.Sprintf(`<body style="margin:0;padding:0;background:%s;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">`, pageBG))
	b.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;"><tr><td align="center" style="padding:32px 16px;">`)
	b.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:560px;border-collapse:collapse;">`)

	b.WriteString(`<tr><td align="center" style="padding-bottom:20px;">`)
	b.WriteString(fmt.Sprintf(`<span style="font-size:22px;font-weight:700;letter-spacing:-0.02em;color:%s;">smartfind</span>`, moss))
	b.WriteString(`</td></tr>`)

	b.WriteString(`<tr><td align="center" style="padding-bottom:8px;">`)
	b.WriteString(fmt.Sprintf(`<span style="display:block;font-size:26px;font-weight:800;line-height:1.15;letter-spacing:0.04em;color:%s;text-transform:uppercase;">We may have found</span>`, moss))
	b.WriteString(fmt.Sprintf(`<span style="display:block;font-size:26px;font-weight:800;line-height:1.15;letter-spacing:0.04em;color:%s;text-transform:uppercase;">A match for you</span>`, accent))
	b.WriteString(`</td></tr>`)

	if len(matches) > 0 && len(matches[0].ImageURLs) > 0 {
		u := html.EscapeString(matches[0].ImageURLs[0])
		alt := html.EscapeString(safe(matches[0].ItemName, "Found item"))
		b.WriteString(`<tr><td align="center" style="padding:20px 0 8px;">`)
		b.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" width="320" style="display:block;max-width:92%%;height:auto;border-radius:16px;border:1px solid #c8e6c9;box-shadow:0 4px 24px rgba(0,0,0,0.06);" />`, u, alt))
		b.WriteString(`</td></tr>`)
	}

	b.WriteString(fmt.Sprintf(`<tr><td style="background:%s;border-radius:20px;padding:28px 24px 24px;margin-top:8px;">`, panel))

	b.WriteString(fmt.Sprintf(`<p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#1b1b1b;">Hey %s,</p>`, greet))
	b.WriteString(`<p style="margin:0 0 20px;font-size:15px;line-height:1.55;color:#333;text-align:center;">`)
	b.WriteString(fmt.Sprintf(`To help reunite you with <strong style="color:#1b1b1b;">%s</strong>, we found one or more similar items logged by transit staff. `+
		`Review the photos below and open SmartFind to file a claim if one of them is yours.`, itemEsc))
	b.WriteString(`</p>`)

	b.WriteString(`<table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 auto 24px;border-collapse:collapse;"><tr><td align="center" style="border-radius:999px;background:` + accent + `;">`)
	escLink := html.EscapeString(link)
	b.WriteString(fmt.Sprintf(`<a href="%s" style="display:inline-block;padding:14px 28px;font-size:14px;font-weight:700;letter-spacing:0.06em;color:#ffffff;text-decoration:none;text-transform:uppercase;">View matches in SmartFind</a>`, escLink))
	b.WriteString(`</td></tr></table>`)

	for i, m := range matches {
		name := html.EscapeString(safe(m.ItemName, "Item"))
		scorePct := int(m.SimilarityScore*100 + 0.5)
		if scorePct > 100 {
			scorePct = 100
		}
		b.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="margin-bottom:14px;border-collapse:collapse;background:#ffffff;border-radius:14px;overflow:hidden;border:1px solid #c8e6c9;">`)
		b.WriteString(`<tr><td style="padding:14px 16px;">`)
		b.WriteString(fmt.Sprintf(`<p style="margin:0 0 6px;font-size:13px;font-weight:700;color:%s;text-transform:uppercase;letter-spacing:0.08em;">Match %d</p>`, moss, i+1))
		b.WriteString(fmt.Sprintf(`<p style="margin:0 0 10px;font-size:16px;font-weight:600;color:#1b1b1b;">%s</p>`, name))
		b.WriteString(fmt.Sprintf(`<p style="margin:0 0 12px;font-size:12px;color:#666;">Similarity score: <strong>%d%%</strong></p>`, scorePct))

		if len(m.ImageURLs) > 0 {
			b.WriteString(`<table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;"><tr>`)
			for _, raw := range m.ImageURLs {
				u := html.EscapeString(raw)
				b.WriteString(`<td style="padding:4px;vertical-align:top;">`)
				b.WriteString(fmt.Sprintf(`<img src="%s" alt="" width="120" style="display:block;width:120px;max-width:100%%;height:auto;border-radius:10px;border:1px solid #e0e0e0;" />`, u))
				b.WriteString(`</td>`)
			}
			b.WriteString(`</tr></table>`)
		} else {
			b.WriteString(`<p style="margin:0;font-size:12px;color:#888;font-style:italic;">No photos on file for this item.</p>`)
		}
		b.WriteString(`</td></tr></table>`)
	}

	b.WriteString(fmt.Sprintf(`<p style="margin:16px 0 0;font-size:11px;line-height:1.5;color:#555;text-align:center;">Lost report ID: <strong>%s</strong><br/>`, html.EscapeString(lostReportID)))
	b.WriteString(`Links and inline photos use time-limited secure URLs — open this email soon for the best experience.</p>`)

	b.WriteString(`</td></tr>`)

	b.WriteString(`<tr><td align="center" style="padding:20px 8px 0;font-size:11px;line-height:1.5;color:#777;">`)
	b.WriteString(`You received this because you filed a lost-item report with SmartFind.<br/>`)
	b.WriteString(`If you did not request this, you can ignore this message.</td></tr>`)

	b.WriteString(`</table></td></tr></table></body></html>`)
	return b.String()
}

func safe(s string, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}
