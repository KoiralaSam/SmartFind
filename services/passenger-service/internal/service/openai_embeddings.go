package service

import (
	"context"

	"smartfind/services/passenger-service/internal/core/ports/inbound"
	"smartfind/shared/embedtext"
	"smartfind/shared/openai"
)

func buildLostReportEmbeddingText(in inbound.CreateLostReportInput) string {
	return embedtext.JoinNonEmpty([]embedtext.Pair{
		{Slot: embedtext.SlotItemName, Value: in.ItemName},
		{Slot: embedtext.SlotItemDescription, Value: in.ItemDescription},
		{Slot: embedtext.SlotItemType, Value: in.ItemType},
		{Slot: embedtext.SlotBrand, Value: in.Brand},
		{Slot: embedtext.SlotModel, Value: in.Model},
		{Slot: embedtext.SlotColor, Value: in.Color},
		{Slot: embedtext.SlotMaterial, Value: in.Material},
		{Slot: embedtext.SlotItemCondition, Value: in.ItemCondition},
		{Slot: embedtext.SlotCategory, Value: in.Category},
		{Slot: embedtext.SlotLocation, Value: in.LocationLost},
		{Slot: embedtext.SlotRoute, Value: in.RouteOrStation},
		{Slot: embedtext.SlotRouteID, Value: in.RouteID},
	})
}

func embedLostReportOpenAI(ctx context.Context, in inbound.CreateLostReportInput) ([]float32, error) {
	return openai.EmbedText(ctx, buildLostReportEmbeddingText(in))
}
