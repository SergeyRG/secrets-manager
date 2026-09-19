package orchestrator

import (
	"context"
	"errors"

	authUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases"
	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	securityUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
)

type LoginOrchestrator struct {
	loginUC  *authUsecases.LoginUseCase
	getKeyUC *securityUsecases.GetUserKeyUseCase
	crKeyUC  *securityUsecases.CreateKeyUseCase
}

func NewLoginOrchestrator(
	loginUC *authUsecases.LoginUseCase,
	crKeyUC *securityUsecases.CreateKeyUseCase,
	getKeyUC *securityUsecases.GetUserKeyUseCase,
) *LoginOrchestrator {
	return &LoginOrchestrator{loginUC: loginUC, crKeyUC: crKeyUC, getKeyUC: getKeyUC}
}

func (o *LoginOrchestrator) Execute(
	ctx context.Context,
	ts authUsecases.TokenStorage,
	ks *domain.KeyStorage,
	pp securityUsecases.PasswdProvider,
) error {
	err := o.loginUC.Execute(ctx, ts)
	if err != nil {
		return err
	}

	err = o.getKeyUC.Execute(ctx, ks, pp)
	if err == nil {
		return nil
	}
	if errors.Is(securityUsecases.ErrKeyNotExist, err) {
		err = o.crKeyUC.Execute(ctx, pp, ks)
		if err != nil {
			return err
		}
	}

	return err
}
