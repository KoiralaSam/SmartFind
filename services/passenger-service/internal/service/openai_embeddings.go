package service

import (
	"context"
	"strings"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/shared/openai"
)

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

func embedLostReportOpenAI(ctx context.Context, in inbound.CreateLostReportInput) ([]float32, error) {
	return openai.EmbedText(ctx, buildLostReportEmbeddingText(in))
}
