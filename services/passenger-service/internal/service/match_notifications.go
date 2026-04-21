package service

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

	"smartfind/shared/env"
	"smartfind/shared/s3media"
)

type MatchCandidate struct {
	FoundItemID     string
	ItemName        string
	SimilarityScore float64
	PrimaryImageKey string
	ImageKeys       []string
}

type MatchReportContext struct {
	LostReportID   string
	PassengerID    string
	PassengerEmail string
	LostItemName   string
	LastEmailedAt  *time.Time
}

type MatchNotificationStore interface {
	CountNotificationsForLostReportSince(ctx context.Context, lostReportID string, window time.Duration) (int, error)
	CountNotificationsForPassengerSince(ctx context.Context, passengerID string, window time.Duration) (int, error)
	InsertMatchNotification(ctx context.Context, passengerID, lostReportID, foundItemID string, similarity float64, itemName string, imageKeys []string, primaryImageKey string) (bool, error)
	UpdateLostReportMatchAudit(ctx context.Context, lostReportID string, checkedAt *time.Time, emailedAt *time.Time) error
}

type emailMatch struct {
	MatchCandidate
	ImageURLs []string
}

func ApplyMatchNotifications(ctx context.Context, store MatchNotificationStore, report MatchReportContext, matches []MatchCandidate) ([]MatchCandidate, error) {
	if store == nil {
		return nil, errors.New("match notification store is nil")
	}
	if strings.TrimSpace(report.LostReportID) == "" || strings.TrimSpace(report.PassengerID) == "" {
		return nil, errors.New("lost report and passenger ids are required")
	}

	now := time.Now()
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

	reportCount, err := store.CountNotificationsForLostReportSince(ctx, report.LostReportID, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	passengerCount, err := store.CountNotificationsForPassengerSince(ctx, report.PassengerID, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	remaining := minInt(maxPerReport-reportCount, maxPerPassenger-passengerCount)

	inserted := make([]MatchCandidate, 0, len(matches))
	if remaining > 0 {
		for _, match := range matches {
			if remaining <= 0 {
				break
			}
			ok, err := store.InsertMatchNotification(
				ctx,
				report.PassengerID,
				report.LostReportID,
				match.FoundItemID,
				match.SimilarityScore,
				match.ItemName,
				match.ImageKeys,
				match.PrimaryImageKey,
			)
			if err != nil {
				return inserted, err
			}
			if !ok {
				continue
			}
			inserted = append(inserted, match)
			remaining--
		}
	}

	if err := store.UpdateLostReportMatchAudit(ctx, report.LostReportID, &now, nil); err != nil {
		return inserted, err
	}

	if len(inserted) == 0 {
		return inserted, nil
	}
	if report.LastEmailedAt != nil && report.LastEmailedAt.After(now.Add(-time.Duration(emailCooldownMin)*time.Minute)) {
		return inserted, nil
	}
	if err := sendMatchEmail(ctx, report.PassengerEmail, report.LostReportID, report.LostItemName, inserted, emailMaxItems); err != nil {
		log.Printf("match email failed passenger_id=%s lost_report_id=%s: %v", report.PassengerID, report.LostReportID, err)
		return inserted, nil
	}
	if err := store.UpdateLostReportMatchAudit(ctx, report.LostReportID, nil, &now); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func sendMatchEmail(ctx context.Context, toEmail string, lostReportID string, lostItemName string, matches []MatchCandidate, maxItems int) error {
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
	if apiToken == "" || strings.EqualFold(apiToken, "REPLACE_ME") || fromEmail == "" || strings.EqualFold(fromEmail, "REPLACE_ME") {
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

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
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

func enrichMatchesForEmail(ctx context.Context, matches []MatchCandidate) []emailMatch {
	p, err := s3media.GetPresigner(ctx)
	if err != nil || p == nil {
		log.Printf("match email: presigner unavailable, sending without inline images: %v", err)
		out := make([]emailMatch, 0, len(matches))
		for _, m := range matches {
			out = append(out, emailMatch{MatchCandidate: m})
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
		out = append(out, emailMatch{MatchCandidate: m, ImageURLs: urls})
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
	var b strings.Builder
	b.WriteString(`<!doctype html><html><body style="font-family:Arial,sans-serif;color:#111;">`)
	b.WriteString(`<h2 style="margin-bottom:8px;">SmartFind found potential matches</h2>`)
	b.WriteString(`<p style="margin-top:0;">Hi ` + greet + `,</p>`)
	b.WriteString(`<p>We found potential matches for <strong>` + itemEsc + `</strong>.</p>`)
	b.WriteString(`<p style="color:#555;font-size:14px;">Lost report ID: <code>` + html.EscapeString(lostReportID) + `</code></p>`)
	for i, m := range matches {
		b.WriteString(`<div style="border:1px solid #ddd;border-radius:12px;padding:16px;margin:16px 0;">`)
		b.WriteString(`<p style="margin:0 0 8px 0;"><strong>` + html.EscapeString(safe(m.ItemName, "Item")) + `</strong>`)
		b.WriteString(fmt.Sprintf(` <span style="color:#555;">(similarity: %.0f%%)</span></p>`, m.SimilarityScore*100))
		if len(m.ImageURLs) > 0 {
			b.WriteString(`<div style="display:flex;gap:8px;flex-wrap:wrap;">`)
			for _, u := range m.ImageURLs {
				esc := html.EscapeString(u)
				b.WriteString(`<a href="` + esc + `" target="_blank" rel="noopener noreferrer">`)
				b.WriteString(`<img src="` + esc + `" alt="Possible match ` + fmt.Sprintf(`%d`, i+1) + `" style="width:140px;height:140px;object-fit:cover;border-radius:10px;border:1px solid #ddd;" />`)
				b.WriteString(`</a>`)
			}
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`<p><a href="` + html.EscapeString(link) + `" style="display:inline-block;background:#111;color:#fff;padding:10px 14px;border-radius:8px;text-decoration:none;">Review matches in SmartFind</a></p>`)
	b.WriteString(`<p style="color:#666;font-size:13px;">Links and inline photos use time-limited secure URLs — open this email soon for the best experience.</p>`)
	b.WriteString(`</body></html>`)
	return b.String()
}

func safe(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
