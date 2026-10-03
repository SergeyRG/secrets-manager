package usecases

import (
	"context"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
)

type AuthUserUseCase struct {
	repo   UserRepo
	hasher Hasher
}

func NewAuthUserUseCase(r UserRepo, h Hasher) *AuthUserUseCase {
	return &AuthUserUseCase{repo: r, hasher: h}
}

func (uc *AuthUserUseCase) Execute(ctx context.Context, login string, pwd string) (domain.UserID, error) {
	user, err := uc.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return "", err
	}

	if !uc.hasher.CheckPasswordHash(pwd, user.PwdHash) {
		return "", ErrWrongLoginOrPassword
	}

	return user.UserID, nil
}
