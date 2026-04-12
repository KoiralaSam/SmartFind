package googleauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Profile holds identity fields from a verified Google ID token.
type Profile struct {
	Email      string
	FullName   string
	PictureURL string
}

type tokenInfoResponse struct {
	Aud           string `json:"aud"`
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Error         string `json:"error"`
	ErrorDesc     string `json:"error_description"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// VerifyIDToken validates the credential via Google's tokeninfo and checks audience + email.
func VerifyIDToken(ctx context.Context, idToken string, expectedAudience string) (*Profile, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, errors.New("id_token is required")
	}
	expectedAudience = strings.TrimSpace(expectedAudience)
	if expectedAudience == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID is required")
	}

	u, err := url.Parse("https://oauth2.googleapis.com/tokeninfo")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("id_token", idToken)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google tokeninfo: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var ti tokenInfoResponse
	if err := json.Unmarshal(body, &ti); err != nil {
		return nil, err
	}
	if ti.Error != "" {
		return nil, fmt.Errorf("google tokeninfo: %s: %s", ti.Error, ti.ErrorDesc)
	}
	if ti.Aud != expectedAudience {
		return nil, errors.New("id token audience does not match GOOGLE_CLIENT_ID")
	}
	if !emailVerified(ti.EmailVerified) {
		return nil, errors.New("google email is not verified")
	}
	email := strings.TrimSpace(strings.ToLower(ti.Email))
	if email == "" {
		return nil, errors.New("google token has no email")
	}
	return &Profile{
		Email:      email,
		FullName:   strings.TrimSpace(ti.Name),
		PictureURL: strings.TrimSpace(ti.Picture),
	}, nil
}

func emailVerified(v any) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true")
	default:
		return false
	}
}
