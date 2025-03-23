package service

import (
	"context"
	"gRPC/internal/repo"
	"gRPC/pkg/secure"
	"gRPC/pkg/validator"
	sso "gRPC/proto"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if err := validator.Validate(ctx, req); err != nil {
		s.log.Errorf("validation error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	passwordValidityCheck, err := secure.IsValidPassword(req.Password)

	if !passwordValidityCheck {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	req.Password, _ = secure.HashPassword(req.Password)

	_, err = s.repo.Register(ctx, &repo.User{
		Username:       req.GetUsername(),
		HashedPassword: req.GetPassword(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	return &sso.RegisterResponse{}, nil
}

func (s *authService) Login(ctx context.Context, req *sso.LoginRequest) (*sso.LoginResponse, error) {
	panic("don't implement")
}
