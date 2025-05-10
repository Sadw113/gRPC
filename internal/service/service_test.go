package service

import (
	"context"
	"errors"
	sso "gRPC/gRPC/genGo"
	"gRPC/internal/config"
	"gRPC/internal/repo"
	"gRPC/internal/repo/mocks"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepo struct {
	repo *mocks.Repository
}

func setupAuthServer(*testing.T) (*authService, *mockRepo) {
	cfg := config.AppConfig{
		System: config.System{
			LockPasswordEntry: 5,
		},
	}

	mock := &mockRepo{
		repo: &mocks.Repository{},
	}

	server := &authService{
		cfg:                   cfg,
		repo:                  mock.repo,
		numberPasswordEntries: cache.New(cfg.System.LockPasswordEntry, cfg.System.LockPasswordEntry),
	}

	return server, mock
}

func TestRegister(t *testing.T) {
	t.Run("регистрация прошла успешно", func(t *testing.T) {
		server, deps := setupAuthServer(t)

		req := &sso.RegisterRequest{
			Username: "User3",
			Password: "User3Password01,)",
		}

		deps.repo.On("GetUser", mock.Anything, req.Username).
			Return(nil, errors.New("user not found")).
			Once()

		deps.repo.On("CreateUser", mock.Anything, mock.MatchedBy(func(user *repo.User) bool {
			return user.Username == req.Username && user.HashedPassword != ""
		})).
			Return(&repo.User{Username: req.Username}, nil).
			Once()

		resp, err := server.Register(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		deps.repo.AssertExpectations(t)
	})
}
