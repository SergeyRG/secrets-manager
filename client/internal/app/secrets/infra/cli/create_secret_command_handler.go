package cli

import (
	"context"
	"errors"
	"fmt"

	orch "github.com/SergeyRG/secrets-manager/client/internal/app/orchestrators"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type CreateSecretCommandHandler struct {
	createSecretOrch *orch.CreateSecretOrch
	getters          map[domain.SecretType]SecretDataGetter
}

func NewCreateSecretDataCommandHandler(orch *orch.CreateSecretOrch, getters map[domain.SecretType]SecretDataGetter) *CreateSecretCommandHandler {
	return &CreateSecretCommandHandler{createSecretOrch: orch, getters: getters}
}

func (h *CreateSecretCommandHandler) Handle(ctx context.Context, args []string, prompter sharedCli.Prompter) error {
	if len(args) != 1 {
		return sharedCli.ErrInvalidArguments
	}

	secretName := args[0]

	if secretName == "" {
		return sharedCli.ErrInvalidArguments
	}
	err := prompter.Send(`Выберите тип секрета:
1 - Произвольные бинарные данные
2 - Произвольный текст
3 - Логин и пароль
4 - Данные банковской карты
: `)
	if err != nil {
		return errors.New("ошибка вывода данных")
	}
	secretTypeStr, err := prompter.Receive(false)

	if err != nil {
		return errors.New("ошибка ввода данных")
	}
	var secretType domain.SecretType
	switch secretTypeStr {
	case "1":
		secretType = domain.SecretTypeBinary
	case "2":
		secretType = domain.SecretTypeFreeText
	case "3":
		secretType = domain.SecretTypeAuthData
	case "4":
		secretType = domain.SecretTypeBankCard
	}

	data, err := h.getters[secretType].GetDataFromUser(prompter)
	if err != nil {
		return err
	}

	err = h.createSecretOrch.Execute(ctx, secretName, secretType, data)
	if err != nil {
		return fmt.Errorf("ошибка получения данных секрета: %w", err)
	}

	return nil
}
