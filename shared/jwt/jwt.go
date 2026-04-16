package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"smartfind/shared/env"
)

// Claims is a minimal JWT claim set for sessions.
// sub: subject (user id)
// email: user email
// exp: expiration (unix seconds)
type Claims struct {
	ActorType string `json:"actorType,omitempty"`
	Sub       string `json:"sub"`
	Email     string `json:"email"`
	Exp       int64  `json:"exp"`
}

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrInvalidSig     = errors.New("invalid token signature")
	ErrExpired        = errors.New("token expired")
	ErrMissingSubject = errors.New("missing subject")
	ErrMissingSecret  = errors.New("missing JWT_SECRET")
)

func secretFromEnv() (string, error) {
	secret := strings.TrimSpace(env.GetString("JWT_SECRET", ""))
	if secret == "" {
		return "", ErrMissingSecret
	}
	return secret, nil
}

func ttlFromEnv() time.Duration {
	// Optional env override. Default: 7 days.
	// JWT_TTL_SECONDS should be an integer number of seconds.
	secs := env.GetInt("JWT_TTL_SECONDS", 0)
	if secs <= 0 {
		return 7 * 24 * time.Hour
	}
	return time.Duration(secs) * time.Second
}

// GenerateToken creates a compact JWT string using HS256.
func GenerateToken(secret string, claims Claims) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("jwt secret is required")
	}
	if strings.TrimSpace(claims.Sub) == "" {
		return "", ErrMissingSubject
	}
	if claims.Exp == 0 {
		return "", errors.New("exp is required")
	}

	headerJSON := []byte(`{"alg":"HS256","typ":"JWT"}`)
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	header := enc.EncodeToString(headerJSON)
	payload := enc.EncodeToString(payloadJSON)
	signingInput := header + "." + payload

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	sig := enc.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}

// GenerateUserToken creates a session JWT from user id + email using JWT_SECRET from the environment.
func GenerateUserToken(userID string, email string) (string, error) {
	secret, err := secretFromEnv()
	if err != nil {
		return "", err
	}
	ttl := ttlFromEnv()
	if strings.TrimSpace(userID) == "" {
		return "", ErrMissingSubject
	}
	if strings.TrimSpace(email) == "" {
		return "", errors.New("email is required")
	}
	return GenerateToken(secret, Claims{
		Sub:   userID,
		Email: strings.TrimSpace(strings.ToLower(email)),
		Exp:   time.Now().Add(ttl).Unix(),
	})
}

func GenerateActorToken(actorType string, userID string, email string) (string, error) {
	secret, err := secretFromEnv()
	if err != nil {
		return "", err
	}
	ttl := ttlFromEnv()
	if strings.TrimSpace(userID) == "" {
		return "", ErrMissingSubject
	}
	if strings.TrimSpace(email) == "" {
		return "", errors.New("email is required")
	}
	if strings.TrimSpace(actorType) == "" {
		return "", errors.New("actorType is required")
	}
	return GenerateToken(secret, Claims{
		ActorType: strings.TrimSpace(actorType),
		Sub:       userID,
		Email:     strings.TrimSpace(strings.ToLower(email)),
		Exp:       time.Now().Add(ttl).Unix(),
	})
}

func GenerateStaffToken(staffID string, email string) (string, error) {
	return GenerateActorToken("staff", staffID, email)
}

func GeneratePassengerToken(passengerID string, email string) (string, error) {
	return GenerateActorToken("passenger", passengerID, email)
}

// VerifyHS256 verifies signature + exp and returns claims.
func VerifyToken(secret string, token string) (Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return Claims{}, errors.New("jwt secret is required")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}
	headerB64, payloadB64, sigB64 := parts[0], parts[1], parts[2]
	if headerB64 == "" || payloadB64 == "" || sigB64 == "" {
		return Claims{}, ErrInvalidToken
	}

	enc := base64.RawURLEncoding
	wantSig, err := enc.DecodeString(sigB64)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	if !hmac.Equal(mac.Sum(nil), wantSig) {
		return Claims{}, ErrInvalidSig
	}

	rawPayload, err := enc.DecodeString(payloadB64)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var c Claims
	if err := json.Unmarshal(rawPayload, &c); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if strings.TrimSpace(c.Sub) == "" {
		return Claims{}, ErrMissingSubject
	}
	if strings.TrimSpace(c.Email) == "" {
		return Claims{}, errors.New("missing email")
	}
	if c.Exp != 0 && time.Now().Unix() > c.Exp {
		return Claims{}, ErrExpired
	}
	return c, nil
}

// VerifyTokenFromEnv verifies token using JWT_SECRET from environment.
func VerifyTokenFromEnv(token string) (Claims, error) {
	secret, err := secretFromEnv()
	if err != nil {
		return Claims{}, err
	}
	return VerifyToken(secret, token)
}
