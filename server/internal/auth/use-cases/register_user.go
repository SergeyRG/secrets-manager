package usecases

import (
	"context"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	authDomain "github.com/SergeyRG/secrets-manager/server/internal/auth/domain"
)

type RegisterUserUseCase struct {
	repo   UserRepo
	hasher Hasher
}

func NewRegisterUserUseCase(r UserRepo, h Hasher) *RegisterUserUseCase {
	return &RegisterUserUseCase{repo: r, hasher: h}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, login string, password string) (domain.UserID, error) {
	passwordHash, err := uc.hasher.HashPassword(password)
	if err != nil {
		return "", err
	}
	user, err := authDomain.NewUser(login, passwordHash)
	if err != nil {
		return "", err
	}
	err = uc.repo.AddUser(ctx, user)
	if err != nil {
		return "", err
	}
	return user.UserID, err
}
