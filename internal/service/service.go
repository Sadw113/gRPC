package service

import (
	"context"
	"fmt"
	"gRPC/internal/repo"
	"gRPC/pkg/jwt"
	"gRPC/pkg/secure"
	"gRPC/pkg/validator"
	sso "gRPC/proto"
	"strconv"

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

	var user repo.User

	user.Username = req.Username
	user.HashedPassword = req.Password

	fmt.Println(user.Username, user.HashedPassword)

	err = s.repo.CreateUser(ctx, &user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	return &sso.RegisterResponse{Message: "Creating new user was successfull"}, nil
}

func (s *authService) Login(ctx context.Context, req *sso.LoginRequest) (*sso.LoginResponse, error) {
	if err := validator.Validate(ctx, req); err != nil {
		s.log.Errorf("validation error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := s.repo.Login(ctx, req.GetUsername())
	if err != nil {
		s.log.Errorf("failed to get credentials for user %s: %v", req.GetUsername(), err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err := secure.CheckPassword(user.HashedPassword, req.GetPassword()); err != nil {
		s.log.Errorf("invalid password for user %s: %v", req.GetUsername(), err)
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	accessToken, err := jwt.GenerateAccessToken(strconv.FormatInt(user.ID, 10))
	if err != nil {
		s.log.Errorf("failed to generate access token for user %s: %v", req.GetUsername(), err)
		return nil, errors.Wrap(err, "failed to generate token")
	}

	return &sso.LoginResponse{Token: accessToken}, nil
}
