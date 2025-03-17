package service

import (
	"context"
	"gRPC/internal/repo"

	"go.uber.org/zap"
)

type AuthService interface {
	Register(ctx context.Context) error
	Login(ctx context.Context) error
}

type authService struct {
	repo repo.Repository
	log  *zap.SugaredLogger
}

func NewService(repo repo.Repository, logger *zap.SugaredLogger) AuthService {
	return &authService{
		repo: repo,
		log:  logger,
	}
}

func (s *authService) Register(ctx context.Context) error {
	// TODO
	return nil
}

func (s *authService) Login(ctx context.Context) error {
	// TODO
	return nil
}
