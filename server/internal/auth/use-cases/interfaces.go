package usecases

import (
	"context"

	authDomain "github.com/SergeyRG/secrets-manager/server/internal/auth/domain"
)

type Hasher interface {
	HashPassword(password string) (string, error)
	CheckPasswordHash(password, hash string) bool
}

type UserRepo interface {
	GetUserByLogin(ctx context.Context, login string) (*authDomain.User, error)
	AddUser(ctx context.Context, user authDomain.User) error
}
