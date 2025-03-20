package service

import (
	"context"
	"gRPC/internal/repo"
	sso "gRPC/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type AuthService interface {
	Register(ctx context.Context, req *sso.RegisterRequest) (*sso.RegisterResponse, error)
	Login(ctx context.Context, req *sso.LoginRequest) (*sso.LoginResponse, error)
}

type authService struct {
	sso.UnimplementedAuthServiceServer
	repo repo.Repository
	log  *zap.SugaredLogger
}

func NewService(repo repo.Repository, logger *zap.SugaredLogger) AuthService {
	return &authService{
		repo: repo,
		log:  logger,
	}
}

func Register(gPRC *grpc.Server) {
	sso.RegisterAuthServiceServer(gPRC, &authService{})
}

func (s *authService) Register(ctx context.Context, req *sso.RegisterRequest) (*sso.RegisterResponse, error) {
	panic("don't implement")
}

func (s *authService) Login(ctx context.Context, req *sso.LoginRequest) (*sso.LoginResponse, error) {
	panic("don't implement")
}
