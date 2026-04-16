package mapper

import (
	"time"

	"smartfind/services/staff-service/internal/core/ports/inbound"
	pb "smartfind/shared/proto/staff"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func StaffToPB(s *inbound.Staff) *pb.Staff {
	if s == nil {
		return nil
	}
	return &pb.Staff{
		Id:        s.ID,
		FullName:  s.FullName,
		Email:     s.Email,
		CreatedAt: timeToTimestamp(s.CreatedAt),
		UpdatedAt: timeToTimestamp(s.UpdatedAt),
	}
}

func FoundItemToPB(it *inbound.FoundItem) *pb.FoundItem {
	if it == nil {
		return nil
	}
	return &pb.FoundItem{
		Id:              it.ID,
		PostedByStaffId: it.PostedByStaffID,
		ItemName:        it.ItemName,
		ItemDescription: it.ItemDescription,
		ItemType:        it.ItemType,
		Brand:           it.Brand,
		Model:           it.Model,
		Color:           it.Color,
		Material:        it.Material,
		ItemCondition:   it.ItemCondition,
		Category:        it.Category,
		LocationFound:   it.LocationFound,
		RouteOrStation:  it.RouteOrStation,
		RouteId:         it.RouteID,
		DateFound:       timeToTimestamp(it.DateFound),
		Status:          it.Status,
		CreatedAt:       timeToTimestamp(it.CreatedAt),
		UpdatedAt:       timeToTimestamp(it.UpdatedAt),
		ImageKeys:       it.ImageKeys,
		PrimaryImageKey: it.PrimaryImageKey,
	}
}

func FoundItemsToPB(items []inbound.FoundItem) []*pb.FoundItem {
	out := make([]*pb.FoundItem, len(items))
	for i := range items {
		out[i] = FoundItemToPB(&items[i])
	}
	return out
}

func ItemClaimToPB(c *inbound.ItemClaim) *pb.ItemClaim {
	if c == nil {
		return nil
	}
	return &pb.ItemClaim{
		Id:                  c.ID,
		ItemId:              c.ItemID,
		ClaimantPassengerId: c.ClaimantPassengerID,
		LostReportId:        c.LostReportID,
		Message:             c.Message,
		Status:              c.Status,
		CreatedAt:           timeToTimestamp(c.CreatedAt),
		UpdatedAt:           timeToTimestamp(c.UpdatedAt),
	}
}

func ItemClaimsToPB(claims []inbound.ItemClaim) []*pb.ItemClaim {
	out := make([]*pb.ItemClaim, len(claims))
	for i := range claims {
		out[i] = ItemClaimToPB(&claims[i])
	}
	return out
}

func RouteToPB(rt *inbound.Route) *pb.Route {
	if rt == nil {
		return nil
	}
	return &pb.Route{
		Id:               rt.ID,
		RouteName:        rt.RouteName,
		CreatedByStaffId: rt.CreatedByStaffID,
		CreatedAt:        timeToTimestamp(rt.CreatedAt),
		UpdatedAt:        timeToTimestamp(rt.UpdatedAt),
	}
}

func RoutesToPB(routes []inbound.Route) []*pb.Route {
	out := make([]*pb.Route, len(routes))
	for i := range routes {
		out[i] = RouteToPB(&routes[i])
	}
	return out
}

func timeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
