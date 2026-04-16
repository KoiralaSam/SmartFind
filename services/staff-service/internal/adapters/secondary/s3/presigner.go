package s3media

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"smartfind/shared/env"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Presigner struct {
	Bucket     string
	Prefix     string
	PutTTL     time.Duration
	GetTTL     time.Duration
	s3Client   *s3.Client
	presignCli *s3.PresignClient
}

type PresignedPut struct {
	Key     string
	URL     string
	Headers map[string]string
}

func tokenHashPrefix(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])[:12]
}

func randomHex(nBytes int) string {
	if nBytes <= 0 {
		nBytes = 6
	}
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		h := sha256.Sum256([]byte(fmt.Sprintf("fallback:%d", time.Now().UnixNano())))
		return hex.EncodeToString(h[:])[:nBytes*2]
	}
	return hex.EncodeToString(buf)
}

func ContentTypeToExt(contentType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		return "jpg", true
	case "image/png":
		return "png", true
	case "image/webp":
		return "webp", true
	default:
		return "", false
	}
}

func LoadPresigner(ctx context.Context) (*Presigner, error) {
	region := strings.TrimSpace(env.GetString("AWS_REGION", ""))
	bucket := strings.TrimSpace(env.GetString("AWS_S3_BUCKET", ""))
	if region == "" || bucket == "" {
		return nil, errors.New("AWS_REGION and AWS_S3_BUCKET are required")
	}
	if strings.EqualFold(bucket, "REPLACE_ME") {
		return nil, errors.New("AWS_S3_BUCKET is not configured (still REPLACE_ME)")
	}

	accessKey := strings.TrimSpace(env.GetString("AWS_ACCESS_KEY_ID", ""))
	secretKey := strings.TrimSpace(env.GetString("AWS_SECRET_ACCESS_KEY", ""))
	sessionToken := strings.TrimSpace(env.GetString("AWS_SESSION_TOKEN", ""))
	if accessKey == "" || secretKey == "" {
		return nil, errors.New("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are required")
	}
	if strings.EqualFold(accessKey, "REPLACE_ME") || strings.EqualFold(secretKey, "REPLACE_ME") {
		return nil, errors.New("AWS credentials are not configured (still REPLACE_ME)")
	}

	prefix := strings.Trim(env.GetString("AWS_S3_PREFIX", "smartfind"), "/")
	if prefix != "" {
		prefix = prefix + "/"
	}

	putTTL := time.Duration(env.GetInt("AWS_S3_PRESIGN_PUT_TTL_SECONDS", 600)) * time.Second
	if putTTL <= 0 {
		putTTL = 10 * time.Minute
	}
	getTTL := time.Duration(env.GetInt("AWS_S3_PRESIGN_GET_TTL_SECONDS", 600)) * time.Second
	if getTTL <= 0 {
		getTTL = 10 * time.Minute
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)),
	)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimSpace(env.GetString("AWS_S3_ENDPOINT", ""))
	forcePathStyle := env.GetBool("AWS_S3_FORCE_PATH_STYLE", false)

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
		}
		o.UsePathStyle = forcePathStyle
	})

	return &Presigner{
		Bucket:     bucket,
		Prefix:     prefix,
		PutTTL:     putTTL,
		GetTTL:     getTTL,
		s3Client:   s3Client,
		presignCli: s3.NewPresignClient(s3Client),
	}, nil
}

var (
	_once     sync.Once
	_instance *Presigner
	_err      error
)

func GetPresigner(ctx context.Context) (*Presigner, error) {
	_once.Do(func() {
		_instance, _err = LoadPresigner(ctx)
	})
	return _instance, _err
}

func (p *Presigner) ObjectKey(environment, sessionToken, ext string, now time.Time) string {
	environment = strings.TrimSpace(environment)
	if environment == "" {
		environment = "development"
	}
	tokenHash := tokenHashPrefix(strings.TrimSpace(sessionToken))
	if tokenHash == "" {
		tokenHash = "anon"
	}
	datePath := fmt.Sprintf("%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	randPart := randomHex(6)
	return fmt.Sprintf("%s%s/sessions/%s/%s/%d-%s.%s",
		p.Prefix,
		environment,
		tokenHash,
		datePath,
		now.UnixMilli(),
		randPart,
		ext,
	)
}

func (p *Presigner) AllowedSessionPrefix(environment, sessionToken string) string {
	environment = strings.TrimSpace(environment)
	if environment == "" {
		environment = "development"
	}
	tokenHash := tokenHashPrefix(strings.TrimSpace(sessionToken))
	if tokenHash == "" {
		tokenHash = "anon"
	}
	return strings.TrimSuffix(p.Prefix, "/") + "/" + environment + "/sessions/" + tokenHash + "/"
}

func (p *Presigner) PresignPut(ctx context.Context, key string) (PresignedPut, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return PresignedPut{}, errors.New("key is required")
	}
	req := &s3.PutObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(key),
	}
	out, err := p.presignCli.PresignPutObject(ctx, req, func(o *s3.PresignOptions) {
		o.Expires = p.PutTTL
	})
	if err != nil {
		return PresignedPut{}, err
	}
	return PresignedPut{
		Key:     key,
		URL:     out.URL,
		Headers: map[string]string{},
	}, nil
}

func (p *Presigner) PresignGet(ctx context.Context, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("key is required")
	}
	req := &s3.GetObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(key),
	}
	out, err := p.presignCli.PresignGetObject(ctx, req, func(o *s3.PresignOptions) {
		o.Expires = p.GetTTL
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func (p *Presigner) DeleteObject(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is required")
	}
	_, err := p.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(key),
	})
	return err
}
