package service

import (
	"context"
	"database/sql"
	sso "gRPC/gRPC/genGo"
	"gRPC/internal/config"
	"gRPC/internal/repo"
	"gRPC/pkg/jwt"
	"gRPC/pkg/secure"
	"gRPC/pkg/validator"
	"strconv"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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
	jwt                   jwt.JWTClient
	numberPasswordEntries *cache.Cache
}

func NewService(repo repo.Repository, logger *zap.SugaredLogger, cfg config.AppConfig, jwt jwt.JWTClient) sso.AuthServiceServer {
	return &authService{
		repo: repo,
		log:  logger,
		cfg:  cfg,
		jwt:  jwt,
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
		s.log.Errorf("Creating a new user with ID %d was failed", id, zap.Error(err))
		return nil, errors.Wrap(err, "failed to create user")
	}

	message := "Creating a new user was successfull"

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

	tokens, _ := s.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: user.ID,
	})

	users_tokens := repo.User_Tokens{
		User_ID:      user.ID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}

	tokens_id, err := s.repo.CreateTokens(ctx, &users_tokens)
	if err != nil {
		s.log.Errorf("failed to insert tokens to DataBase. User ID: %d", tokens_id)
	}

	return &sso.LoginResponse{
		Accesstoken:  tokens.AccessToken,
		Refreshtoken: tokens.RefreshToken}, nil
}

func (s *authService) UpdatePassword(ctx context.Context, req *sso.UpdatePasswordRequest) (*sso.UpdatePasswordResponse, error) {
	user, err := s.repo.GetUser(ctx, req.GetUsername())
	if err != nil {
		s.log.Errorf("failed to get credentials for user %s: %v", req.GetUsername(), err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	remainingAttempts, err := s.checkRemainingAttempts(user.ID)
	if err != nil {
		return nil, err
	}

	passwordValidityCheck, err := secure.IsValidPassword(req.NewPassword)
	if !passwordValidityCheck {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	password, err := s.repo.GetPassword(ctx, req.GetUsername())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get password")
	}

	err = secure.CheckPassword(password, req.Password)

	if req.Password != "" && err != nil {
		s.numberPasswordEntries.Set(strconv.FormatInt(user.ID, 10), remainingAttempts-1, cache.DefaultExpiration)

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

	err = s.repo.UpdatePassword(ctx, req.NewPassword, req.GetUsername())

	if err != nil {
		return nil, errors.Wrap(err, "Failed changing password")
	}

	s.numberPasswordEntries.Delete(strconv.FormatInt(user.ID, 10))

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

func (s *authService) Validate(ctx context.Context, req *sso.ValidateRequest) (*sso.ValidateResponse, error) {
	check, err := s.jwt.ValidateToken(&jwt.ValidateTokenParams{
		Token: req.AccessToken,
	})

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}

	if !check {
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}

	accessData, err := s.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: req.AccessToken,
	})

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}

	_, err = s.repo.GetRefreshToken(ctx, accessData.UserId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, "not authorized")
		}

		return nil, status.Error(codes.Internal, "try it a little later or check the data you entered")
	}

	return &sso.ValidateResponse{
		UserId: accessData.UserId,
	}, nil
}

func (s *authService) NewJwt(ctx context.Context, req *sso.NewJwtRequest) (*sso.NewJwtResponse, error) {
	if err := validator.Validate(ctx, req); err != nil {
		s.log.Errorf("validation error: %v", err)

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tokens, err := s.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: req.UserId,
	})

	if err != nil {
		s.log.Errorf("create tokens err: user_id = %d", req.UserId)
		return nil, status.Error(codes.Internal, "try it a little later or check the data you entered")
	}

	err = s.repo.NewRefreshToken(ctx, repo.NewRefreshTokenParams{
		UserID: req.UserId,
		Token:  tokens.RefreshToken,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return nil, status.Error(codes.NotFound, "User not found")
			}
		}
		s.log.Errorf("adding a token to the database: user_id = %d", req.UserId)
		return nil, status.Error(codes.Internal, "try it a little later or check the data you entered")
	}

	return &sso.NewJwtResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *authService) RevokeJwt(ctx context.Context, req *sso.RevokeJwtRequest) (*sso.RevokeJwtResponse, error) {
	err := s.repo.DeleteRefreshToken(ctx, req.UserId)
	if err != nil {
		s.log.Errorf("remove a token to the database: user_id = %d", req.UserId)
		return nil, status.Error(codes.Internal, "try it a little later or check the data you entered")
	}
	return &sso.RevokeJwtResponse{}, nil
}

func (s *authService) Refresh(ctx context.Context, req *sso.RefreshRequest) (*sso.RefreshResponse, error) {
	check, err := s.jwt.ValidateToken(&jwt.ValidateTokenParams{
		Token: req.RefreshToken,
	})
	if err != nil {
		s.log.Errorf("validate refresh token err")
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}
	if !check {
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}
	accessData, err := s.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: req.AccessToken,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}

	refreshData, err := s.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: req.RefreshToken,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}
	if accessData.UserId != refreshData.UserId {
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}

	rtToken, err := s.repo.GetRefreshToken(ctx, refreshData.UserId)
	if err != nil {
		s.log.Errorf("get refresh token err")
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "refresh token not found")
		}
		return nil, status.Error(codes.Internal, "try it a little later or check the data you entered")
	}

	if len(rtToken) == 0 {
		s.log.Errorf("len(rtToken) == 0")
		return nil, status.Error(codes.NotFound, "refresh token not found")
	}

	if rtToken != req.RefreshToken {
		s.log.Errorf("rtToken != req.RefreshToken")
		return nil, status.Error(codes.Unauthenticated, "not authorized")
	}

	tokens, err := s.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: refreshData.UserId,
	})

	if err != nil {
		s.log.Errorf("create tokens error")
		return nil, status.Error(codes.Internal, "try it a little later or check the data you entered")
	}

	err = s.repo.NewRefreshToken(ctx, repo.NewRefreshTokenParams{
		Token:  tokens.RefreshToken,
		UserID: refreshData.UserId,
	})

	if err != nil {
		s.log.Errorf("update refresh token err")
		return nil, status.Error(codes.Internal, "try it a little later or check the data you entered")
	}

	return &sso.RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}
