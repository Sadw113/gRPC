package service

import (
	"context"
	"fmt"
	sso "gRPC/gRPC/genGo"
	"gRPC/internal/config"
	"gRPC/internal/repo"
	"gRPC/pkg/jwt"
	"gRPC/pkg/secure"
	"gRPC/pkg/validator"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type authService struct {
	sso.UnimplementedAuthServiceServer
	repo                  repo.Repository
	log                   *zap.SugaredLogger
	cfg                   config.AppConfig
	numberPasswordEntries *cache.Cache
}

func NewService(repo repo.Repository, logger *zap.SugaredLogger, cfg config.AppConfig) sso.AuthServiceServer {
	return &authService{
		repo: repo,
		log:  logger,
		cfg:  cfg,
		numberPasswordEntries: cache.New(
			cfg.System.LockPasswordEntry,
			cfg.System.LockPasswordEntry,
		),
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

	_, err = s.repo.GetUser(ctx, user.Username)
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

	user, err := s.repo.GetUser(ctx, req.GetUsername())
	if err != nil {
		s.log.Errorf("failed to get credentials for user %s: %v", req.GetUsername(), err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err := secure.CheckPassword(user.HashedPassword, req.GetPassword()); err != nil {
		s.log.Errorf("invalid password for user %s: %v", req.GetUsername(), err)
		return nil, status.Error(codes.Unauthenticated, "invalid username or password")
	}

	accessToken, err := jwt.GenerateAccessToken(strconv.FormatInt(user.ID, 10), s.cfg.SecretKeys.AccessSecret)
	if err != nil {
		s.log.Errorf("failed to generate access token for user %s: %v", req.GetUsername(), err)
		return nil, errors.Wrap(err, "failed to generate token")
	}
	refreshToken, err := jwt.GenerateRefreshToken(strconv.FormatInt(user.ID, 10), s.cfg.SecretKeys.RefreshSecret)
	if err != nil {
		s.log.Errorf("failed to generate refresh token for user %s: %v", req.GetUsername(), err)
		return nil, errors.Wrap(err, "failed to generate token")
	}

	return &sso.LoginResponse{
		Accesstoken:  accessToken,
		Refreshtoken: refreshToken}, nil
}

func (s *authService) UpdatePassword(ctx context.Context, req *sso.UpdatePasswordRequest) (*sso.UpdatePasswordResponse, error) {
	remainingAttempts, err := s.checkRemainingAttempts(req.UserId)
	if err != nil {
		return nil, err
	}

	passwordValidityCheck, err := secure.IsValidPassword(req.NewPassword)
	if !passwordValidityCheck {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	password, err := s.repo.GetPassword(ctx, req.GetUserId())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get password")
	}

	err = secure.CheckPassword(password, req.Password)

	if req.Password != "" && err != nil {
		s.numberPasswordEntries.Set(strconv.FormatInt(req.UserId, 10), remainingAttempts-1, cache.DefaultExpiration)

		return nil, status.Errorf(
			codes.InvalidArgument,
			"%s %d",
			"Incorrect password. Number of attempts:",
			remainingAttempts-1,
		)
	}

	err = secure.CheckPassword(password, req.NewPassword)

	if err == nil {
		return nil, status.Error(codes.InvalidArgument, "Please enter new password")
	}

	req.NewPassword, _ = secure.HashPassword(req.NewPassword)

	err = s.repo.UpdatePassword(ctx, req.NewPassword, req.GetUserId())

	if err != nil {
		return nil, errors.Wrap(err, "Failed changing password")
	}

	s.numberPasswordEntries.Delete(strconv.FormatInt(req.UserId, 10))

	return &sso.UpdatePasswordResponse{Message: "The password change was successful"}, nil
}

func (s *authService) checkRemainingAttempts(userId int64) (int64, error) {
	remainingAttempts := s.cfg.System.NumberPasswordAttempts
	remainingAttemptsFromCache, expirationTime, ok := s.numberPasswordEntries.GetWithExpiration(strconv.FormatInt(userId, 10))

	if ok && remainingAttemptsFromCache.(int64) == 0 {
		return 0, lockForActionErr(expirationTime)
	}
	if ok {
		remainingAttempts = remainingAttemptsFromCache.(int64)
	}
	return remainingAttempts, nil
}

func lockForActionErr(time time.Time) error {
	err := status.New(codes.Unavailable, "Exceeded the maximum number of attempts.\nTry again at")
	err, _ = err.WithDetails(&sso.Err{
		ExpirationTime: timestamppb.New(time),
	})
	return err.Err()
}
