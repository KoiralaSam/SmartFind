package passengergrpc

import (
	"smartfind/services/passenger-service/internal/adapters/primary/grpc/handler"
	"smartfind/services/passenger-service/internal/adapters/primary/grpc/interceptor"
	"smartfind/services/passenger-service/internal/core/ports/inbound"
	pb "smartfind/shared/proto/passenger"

	"google.golang.org/grpc"
)

func NewServer(usecase inbound.PassengerUsecase) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.Recovery(),
			interceptor.InternalAuth(),
			interceptor.Claims(map[string]bool{
				"/smartfind.passenger.v1.PassengerService/Login": true,
			}),
			interceptor.Logging(),
		),
	)

	pb.RegisterPassengerServiceServer(srv, handler.New(usecase))
	return srv
}
