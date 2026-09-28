package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type GetSecretListCommandHandler struct {
	uc      *usecases.GetSecretsListUseCase
	perPage int
}

func NewGetSecretListCommandHandler(
	uc *usecases.GetSecretsListUseCase,
	perPage int,
) *GetSecretListCommandHandler {
	return &GetSecretListCommandHandler{
		uc:      uc,
		perPage: perPage,
	}
}

func (h *GetSecretListCommandHandler) Handle(ctx context.Context, args []string, prompter sharedCli.Prompter) error {
	if len(args) != 0 {
		return sharedCli.ErrInvalidArguments
	}

	page := 1

OuterLoop:
	for {
		metas, err := h.uc.Execute(ctx, page, h.perPage)

		isLocalCache := errors.Is(err, usecases.ErrDataFromLocalCache)

		if err != nil && !isLocalCache {
			return fmt.Errorf("ошибка получения списка секретов: %w", err)
		}

		if isLocalCache {
			prompter.Send("Ошибка запроса сервера. Данные получены из локального кэша:\n")
		}

		if len(metas) == 0 {
			if page == 1 {
				prompter.Send("Список секретов пуст.\n")
			} else {
				prompter.Send("Список секретов закончился.\n")
			}
			return nil
		}

		for _, m := range metas {
			prompter.Send(fmt.Sprintf("\tКлюч: %s\t версия: %v\t\n", m.SecretName, m.Version))
		}

		if len(metas) < h.perPage {
			return nil
		}

		prompter.Send("\n")
		for {
			prompter.Send("Для загрузки следующей страницы нажмите 'n', для завершения нажмите 'e': ")
			m, err := prompter.Receive(false)
			if err != nil {
				return fmt.Errorf("ошибка во время чтения пользовательского ввода: %w", err)
			}

			if m == "n" {
				page++
				continue OuterLoop
			} else if m == "e" {
				return nil
			}

			prompter.Send("Неверный ввод. ")
		}
	}
}
