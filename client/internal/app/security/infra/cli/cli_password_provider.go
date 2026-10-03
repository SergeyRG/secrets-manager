package cli

import (
	"context"
	"errors"

	sharedInfra "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type CliPasswordProvider struct {
	promter *sharedInfra.ConsolePrompter
}

func NewCliPasswordProvider(promter *sharedInfra.ConsolePrompter) *CliPasswordProvider {
	return &CliPasswordProvider{promter: promter}
}

func (p *CliPasswordProvider) GetPassword(ctx context.Context) (psswd string, err error) {
	p.promter.Send("Введите пароль ключа шифрования: ")
	psswd, err = p.promter.Receive(true)
	if err != nil {
		return "", err
	}
	return psswd, nil
}

func (p *CliPasswordProvider) CreatePassword(ctx context.Context) (psswd string, err error) {
	p.promter.Send("Создание ключа шифрования.\n")
	p.promter.Send("Введите пароль ключа шифрования: ")
	psswd1, err := p.promter.Receive(true)
	if err != nil {
		return "", err
	}
	p.promter.Send("Повторите ввод: ")
	psswd2, err := p.promter.Receive(true)
	if err != nil {
		return "", err
	}
	if psswd1 != psswd2 {
		return "", errors.New("введенные пароли не совпадают")
	}

	return psswd2, nil
}
