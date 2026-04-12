package mapper

import (
	"time"

	"smartfind/services/passenger-service/internal/core/domain"
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

func timeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

