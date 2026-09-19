package usecases

import (
	"context"

	"github.com/SergeyRG/secrets-manager/client/internal/app/registre/domain"
)

type RegistreUserUseCase struct {
	dataProvider     domain.UserDataProvider
	registreProvider domain.RegistreProvider
}

func NewRegistreUserUseCase(
	dataProvider domain.UserDataProvider,
	registreProvider domain.RegistreProvider,
) *RegistreUserUseCase {
	return &RegistreUserUseCase{
		dataProvider:     dataProvider,
		registreProvider: registreProvider,
	}
}

func (uc RegistreUserUseCase) Execute(ctx context.Context) error {
	u, err := uc.dataProvider.GetUserData(ctx)
	if err != nil {
		return nil
	}
	return uc.registreProvider.Registre(ctx, u)
}
