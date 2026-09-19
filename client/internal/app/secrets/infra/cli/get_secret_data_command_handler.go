package cli

import (
	"context"
	"errors"
	"strconv"

	orch "github.com/SergeyRG/secrets-manager/client/internal/app/orchestrators"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

// type textDataResp struct {
// 	SecretName string `json:"secret_name"`
// 	Version    string `json:"version"`
// 	Data       string `json:"data"`
// }

type GetSecretDataCommandHandler struct {
	orch  *orch.GetSecretDataByNameOrch
	views map[domain.SecretType]View
}

func NewGetSecretDataCommandHandler(orch *orch.GetSecretDataByNameOrch, views map[domain.SecretType]View) *GetSecretDataCommandHandler {
	return &GetSecretDataCommandHandler{orch: orch, views: views}
}

func (h *GetSecretDataCommandHandler) Handle(ctx context.Context, args []string, prompter sharedCli.Prompter) error {
	if len(args) > 2 || len(args) == 0 {
		return sharedCli.ErrInvalidArguments
	}

	versionStr := ""
	if len(args) == 1 {
		versionStr = "0"
	} else {
		versionStr = args[1]

	}

	secretName := args[0]

	if secretName == "" || versionStr == "" {
		return sharedCli.ErrInvalidArguments
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return sharedCli.ErrInvalidArguments
	}

	smt, data, err := h.orch.Execute(ctx, secretName, version)
	if err != nil {
		return errors.New("ошибка получения данных секрета")
	}

	h.views[smt.SecretType].ProvideDataToUser(smt, data, prompter)

	return nil
}
