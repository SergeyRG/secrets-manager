package cli

import (
	"context"

	restyAuthClient "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/resty_passwd_auth_client"
	sharedInfra "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type ConsolePasswdCredsProvider struct {
	promter *sharedInfra.ConsolePrompter
}

func NewConsolePasswdCredsProvider(promter *sharedInfra.ConsolePrompter) *ConsolePasswdCredsProvider {
	return &ConsolePasswdCredsProvider{promter: promter}
}

func (p *ConsolePasswdCredsProvider) GetCreds(ctx context.Context) (creds restyAuthClient.PasswdCreds, err error) {
	p.promter.Send("Вход в систему. Для регистрации запустите приложение с ключом '-r'.\n")
	p.promter.Send("Имя пользователя: ")
	login, err := p.promter.Receive(false)
	if err != nil {
		return restyAuthClient.PasswdCreds{}, err
	}

	p.promter.Send("Пароль: ")
	passwd, err := p.promter.Receive(true)
	if err != nil {
		return restyAuthClient.PasswdCreds{}, err
	}
	return restyAuthClient.PasswdCreds{
		Login:  login,
		Passwd: passwd,
	}, nil
}
