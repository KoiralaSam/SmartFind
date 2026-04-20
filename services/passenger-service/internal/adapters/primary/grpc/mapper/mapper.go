package mapper

import (
	"time"

	"smartfind/services/passenger-service/internal/core/domain"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	pb "smartfind/shared/proto/passenger"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func PassengerToPB(p *domain.Passenger) *pb.Passenger {
	if p == nil {
		return nil
	}
	return &pb.Passenger{
		Id:        p.ID,
		Email:     p.Email,
		FullName:  p.FullName,
		Phone:     p.Phone,
		CreatedAt: timeToTimestamp(p.CreatedAt),
		UpdatedAt: timeToTimestamp(p.UpdatedAt),
		AvatarUrl: p.AvatarURL,
	}
}

func LostReportToPB(r *inbound.LostReport) *pb.LostReport {
	if r == nil {
		return nil
	}
	return &pb.LostReport{
		Id:                  r.ID,
		ReporterPassengerId: r.ReporterPassengerID,
		ItemName:            r.ItemName,
		ItemDescription:     r.ItemDescription,
		ItemType:            r.ItemType,
		Brand:               r.Brand,
		Model:               r.Model,
		Color:               r.Color,
		Material:            r.Material,
		ItemCondition:       r.ItemCondition,
		Category:            r.Category,
		LocationLost:        r.LocationLost,
		RouteOrStation:      r.RouteOrStation,
		RouteId:             r.RouteID,
		DateLost:            timeToTimestamp(r.DateLost),
		Status:              r.Status,
		CreatedAt:           timeToTimestamp(r.CreatedAt),
		UpdatedAt:           timeToTimestamp(r.UpdatedAt),
	}
}

func LostReportsToPB(reports []inbound.LostReport) []*pb.LostReport {
	out := make([]*pb.LostReport, len(reports))
	for i := range reports {
		out[i] = LostReportToPB(&reports[i])
	}
	return out
}

func FoundItemMatchToPB(m *inbound.FoundItemMatch) *pb.FoundItemMatch {
	if m == nil {
		return nil
	}
	return &pb.FoundItemMatch{
		FoundItemId:     m.FoundItemID,
		ItemName:        m.ItemName,
		ItemDescription: m.ItemDescription,
		ItemType:        m.ItemType,
		Brand:           m.Brand,
		Model:           m.Model,
		Color:           m.Color,
		Material:        m.Material,
		ItemCondition:   m.ItemCondition,
		Category:        m.Category,
		LocationFound:   m.LocationFound,
		RouteOrStation:  m.RouteOrStation,
		RouteId:         m.RouteID,
		DateFound:       timeToTimestamp(m.DateFound),
		Status:          m.Status,
		SimilarityScore: m.SimilarityScore,
		ImageUrls:       m.ImageURLs,
		PrimaryImageUrl: m.PrimaryImageURL,
	}
}

func FoundItemMatchesToPB(matches []inbound.FoundItemMatch) []*pb.FoundItemMatch {
	out := make([]*pb.FoundItemMatch, len(matches))
	for i := range matches {
		out[i] = FoundItemMatchToPB(&matches[i])
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

func ItemClaimsToPB(c []inbound.ItemClaim) []*pb.ItemClaim {
	out := make([]*pb.ItemClaim, len(c))
	for i := range c {
		out[i] = ItemClaimToPB(&c[i])
	}
	return out
}

func PassengerMatchNotificationToPB(n *inbound.PassengerMatchNotification) *pb.PassengerMatchNotification {
	if n == nil {
		return nil
	}
	var readAt *timestamppb.Timestamp
	if !n.ReadAt.IsZero() {
		readAt = timeToTimestamp(n.ReadAt)
	}
	return &pb.PassengerMatchNotification{
		Id:              n.ID,
		PassengerId:     n.PassengerID,
		LostReportId:    n.LostReportID,
		FoundItemId:     n.FoundItemID,
		SimilarityScore: n.SimilarityScore,
		ItemName:        n.ItemName,
		ImageUrls:       n.ImageURLs,
		PrimaryImageUrl: n.PrimaryImageURL,
		CreatedAt:       timeToTimestamp(n.CreatedAt),
		ReadAt:          readAt,
	}
}

func PassengerMatchNotificationsToPB(n []inbound.PassengerMatchNotification) []*pb.PassengerMatchNotification {
	out := make([]*pb.PassengerMatchNotification, len(n))
	for i := range n {
		out[i] = PassengerMatchNotificationToPB(&n[i])
	}
	return out
}

func timeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
