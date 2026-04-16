package staffgrpc

import (
	"smartfind/services/staff-service/internal/adapters/primary/grpc/handler"
	"smartfind/services/staff-service/internal/adapters/primary/grpc/interceptor"
	"smartfind/services/staff-service/internal/core/ports/inbound"
	pb "smartfind/shared/proto/staff"

	"google.golang.org/grpc"
)

func NewServer(usecase inbound.StaffUsecase) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.Recovery(),
			interceptor.InternalAuth(),
			interceptor.Claims(map[string]bool{
				"/smartfind.staff.v1.StaffService/Login":                             true,
				"/smartfind.staff.v1.StaffService/CreateStaff":                       true,
				"/smartfind.staff.v1.StaffService/SearchFoundItemMatchesByEmbedding": true,
			}),
			interceptor.Logging(),
		),
	)

	pb.RegisterStaffServiceServer(srv, handler.New(usecase))
	return srv
}
