package usecases

import (
	"context"
	"errors"
)

var (
	ErrAuthenticationFailed = errors.New("ошибка аутентификации")
)

type LoginUseCase struct {
	authClient AuthClient
}

func NewLoginUseCase(authClient AuthClient) *LoginUseCase {
	return &LoginUseCase{authClient: authClient}
}

func (uc LoginUseCase) Execute(ctx context.Context, ts TokenStorage) error {
	err := uc.authClient.Authenticate(ctx, ts)
	if err != nil {
		return err
	}

	return nil
}
