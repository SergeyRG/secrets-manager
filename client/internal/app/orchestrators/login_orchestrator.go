package orchestrator

// import (
// 	"context"
// 	"errors"

// 	authUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases"
// 	securityCacheInfra "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/cache"
// 	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
// 	securityUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
// )

// type GetKeyOrchestrator struct {
// 	loginUC    *authUsecases.LoginUseCase
// 	getKeyUC   *securityUsecases.GetUserKeyUseCase
// 	crKeyUC    *securityUsecases.CreateKeyUseCase
// 	localCache *securityCacheInfra.LocalCache
// }

// func NewGetKeyOrchestrator(
// 	crKeyUC *securityUsecases.CreateKeyUseCase,
// 	getKeyUC *securityUsecases.GetUserKeyUseCase,
// 	localCache *securityCacheInfra.LocalCache,
// ) *LoginOrchestrator {
// 	return &LoginOrchestrator{crKeyUC: crKeyUC, getKeyUC: getKeyUC, localCache: localCache}
// }

// func (o *GetKeyOrchestrator) Execute(
// 	ctx context.Context,
// 	ts authUsecases.TokenStorage,
// 	ks *domain.KeyStorage,
// 	pp securityUsecases.PasswdProvider,
// ) error {

// 	err := o.getKeyUC.Execute(ctx, ks, pp)
// 	if err == nil {
// 		return nil
// 	}
// 	if errors.Is(securityUsecases.ErrKeyNotExist, err) {
// 		err = o.crKeyUC.Execute(ctx, pp, ks)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return err
// }
