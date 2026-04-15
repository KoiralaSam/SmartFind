package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"smartfind/shared/env"
)

const (
	embeddingsURL = "https://api.openai.com/v1/embeddings"

	// text-embedding-3-small returns 1536 dimensions.
	embeddingDims = 1536
)

var defaultHTTPClient = &http.Client{Timeout: 20 * time.Second}

type embeddingsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// EmbedText generates a 1536-dim embedding for the provided text using OpenAI Embeddings.
//
// Env:
// - OPENAI_API_KEY (required)
// - OPENAI_EMBEDDING_MODEL (optional, defaults to text-embedding-3-small)
func EmbedText(ctx context.Context, text string) ([]float32, error) {
	apiKey := strings.TrimSpace(env.GetString("OPENAI_API_KEY", ""))
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required")
	}

	model := strings.TrimSpace(env.GetString("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"))
	if model == "" {
		model = "text-embedding-3-small"
	}

	payload, err := json.Marshal(embeddingsRequest{
		Model: model,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, embeddingsURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out embeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
			return nil, fmt.Errorf("openai embeddings error: %s", out.Error.Message)
		}
		return nil, fmt.Errorf("openai embeddings request failed: status=%d", resp.StatusCode)
	}
	if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
		return nil, fmt.Errorf("openai embeddings error: %s", out.Error.Message)
	}
	if len(out.Data) == 0 {
		return nil, errors.New("openai embeddings: empty data")
	}

	emb64 := out.Data[0].Embedding
	if len(emb64) != embeddingDims {
		return nil, fmt.Errorf("openai embeddings: unexpected dimension %d", len(emb64))
	}

	emb := make([]float32, len(emb64))
	for i, v := range emb64 {
		emb[i] = float32(v)
	}
	return emb, nil
}
