package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/shared/env"
)

const (
	openAIEmbeddingsURL = "https://api.openai.com/v1/embeddings"

	// text-embedding-3-small returns 1536 dimensions.
	openAIEmbeddingDims = 1536
)

type openAIEmbeddingsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func embedTextOpenAI(ctx context.Context, text string) ([]float32, error) {
	apiKey := strings.TrimSpace(env.GetString("OPENAI_API_KEY", ""))
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required")
	}

	model := strings.TrimSpace(env.GetString("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"))
	if model == "" {
		model = "text-embedding-3-small"
	}

	payload, err := json.Marshal(openAIEmbeddingsRequest{
		Model: model,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEmbeddingsURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out openAIEmbeddingsResponse
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
	if len(emb64) != openAIEmbeddingDims {
		return nil, fmt.Errorf("openai embeddings: unexpected dimension %d", len(emb64))
	}

	emb := make([]float32, len(emb64))
	for i, v := range emb64 {
		emb[i] = float32(v)
	}
	return emb, nil
}

func buildLostReportEmbeddingText(in inbound.CreateLostReportInput) string {
	parts := []string{
		in.ItemName,
		in.ItemDescription,
		in.ItemType,
		in.Brand,
		in.Model,
		in.Color,
		in.Material,
		in.ItemCondition,
		in.Category,
		in.LocationLost,
		in.RouteOrStation,
		in.RouteID,
	}

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, " | ")
}
