package repo

import "context"

type repository struct {
	users map[int]User
}

type Repository interface {
	Register(ctx context.Context) error
	Login(ctx context.Context) error
}

func NewRepository(ctx context.Context) (Repository, error) {
	// создаем мапу для хранения данных в памяти
	var users = make(map[int]User)

	return &repository{users: users}, nil
}

func (r *repository) Register(ctx context.Context) error {
	// TODO
	return nil
}

func (r *repository) Login(ctx context.Context) error {
	// TODO
	return nil
}
