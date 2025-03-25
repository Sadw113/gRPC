package service

import (
	"context"
	"fmt"
	sso "gRPC/gRPC/proto"
	"gRPC/internal/repo"
	"gRPC/pkg/jwt"
	"gRPC/pkg/secure"
	"gRPC/pkg/validator"
	"strconv"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authService struct {
	sso.UnimplementedAuthServiceServer
	repo repo.Repository
	log  *zap.SugaredLogger
}

func NewService(repo repo.Repository, logger *zap.SugaredLogger) sso.AuthServiceServer {
	return &authService{
		repo: repo,
		log:  logger,
	}
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

	user := repo.User{
		Username:       req.GetUsername(),
		HashedPassword: req.GetPassword(),
	}

	_, err = s.repo.Login(ctx, user.Username)
	if err == nil {
		s.log.Error("Creating a new user failed: this user is exist", zap.Error(err))
		return &sso.RegisterResponse{Message: "User is exist"}, errors.Wrap(err, "user is exist")
	}

	id, err := s.repo.CreateUser(ctx, &user)
	if err != nil {
		s.log.Error("Creating a new user failed", zap.Error(err))
		return nil, errors.Wrap(err, "failed to create user")
	}

	message := fmt.Sprintf("Creating a new user with id %d was successfull", id)

	return &sso.RegisterResponse{Message: message}, nil
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
	refreshToken, err := jwt.GenerateRefreshToken(strconv.FormatInt(user.ID, 10))
	if err != nil {
		s.log.Errorf("failed to generate refresh token for user %s: %v", req.GetUsername(), err)
		return nil, errors.Wrap(err, "failed to generate token")
	}

	return &sso.LoginResponse{
		Accesstoken:  accessToken,
		Refreshtoken: refreshToken}, nil
}
