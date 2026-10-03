package usecases

import (
	"context"
	"errors"
)

var (
	ErrAuthenticationFailed = errors.New("ошибка аутентификации")
	ErrServerUnavailable    = errors.New("ошибка направления запроса на сервер")
)

type LoginUseCase struct {
	authClient AuthClient
}

func NewLoginUseCase(authClient AuthClient) *LoginUseCase {
	return &LoginUseCase{authClient: authClient}
}

func (uc LoginUseCase) Execute(ctx context.Context, ts TokenStorage) (login string, err error) {
	return uc.authClient.Authenticate(ctx, ts)
}
