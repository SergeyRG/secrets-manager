package cli

import (
	"context"
	"errors"

	"github.com/SergeyRG/secrets-manager/client/internal/app/registre/domain"
	sharedInfra "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type ConsoleDataProvider struct {
	promter *sharedInfra.ConsolePrompter
}

func NewConsoleDataProvider(promter *sharedInfra.ConsolePrompter) *ConsoleDataProvider {
	return &ConsoleDataProvider{promter: promter}
}

func (p *ConsoleDataProvider) GetUserData(ctx context.Context) (domain.User, error) {
	p.promter.Send("Регистрация нового пользователя:\n")
	p.promter.Send("Имя пользователя: ")
	login, err := p.promter.Receive(false)
	if err != nil {
		return domain.User{}, err
	}

	p.promter.Send("Пароль: ")
	passwd1, err := p.promter.Receive(true)
	if err != nil {
		return domain.User{}, err
	}
	p.promter.Send("Повторите пароль: ")
	passwd2, err := p.promter.Receive(true)
	if err != nil {
		return domain.User{}, err
	}
	if passwd1 != passwd2 {
		return domain.User{}, errors.New("введенные пароли не совпадают")
	}

	u, err := domain.NewUser(login, passwd2)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}
