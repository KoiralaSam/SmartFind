package postgres

import (
	"context"
	"log"
	"strings"

	"smartfind/shared/s3media"
)

// presignFoundItemImageURLs turns S3 object keys into short-lived HTTPS URLs.
// If presigning is unavailable, returns empty slices (UI still shows text fields).
func presignFoundItemImageURLs(ctx context.Context, rawKeys []string, rawPrimary string) (urls []string, primaryURL string) {
	presigner, err := s3media.GetPresigner(ctx)
	if err != nil || presigner == nil {
		if err != nil {
			log.Printf("presigner unavailable (%v) — images will not load", err)
		}
		return nil, ""
	}
	urls = make([]string, 0, len(rawKeys))
	for _, k := range rawKeys {
		if strings.TrimSpace(k) == "" {
			continue
		}
		u, err := presigner.PresignGet(ctx, k)
		if err == nil && strings.TrimSpace(u) != "" {
			urls = append(urls, u)
		}
	}
	if strings.TrimSpace(rawPrimary) != "" {
		if u, err := presigner.PresignGet(ctx, rawPrimary); err == nil {
			primaryURL = u
		}
	}
	if primaryURL == "" && len(urls) > 0 {
		primaryURL = urls[0]
	}
	return urls, primaryURL
}
