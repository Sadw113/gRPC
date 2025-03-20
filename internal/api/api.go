package api

import (
	"gRPC/internal/service"

	"google.golang.org/grpc"
)

type Server struct {
	Service service.AuthService
}

func New(s *Server) *grpc.Server {
	gPRCServer := grpc.NewServer()

	service.Register(gPRCServer)

	return gPRCServer
}
