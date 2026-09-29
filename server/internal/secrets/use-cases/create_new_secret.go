package usecases

import (
	"context"
	"errors"
	"io"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	l "github.com/SergeyRG/secrets-manager/internal/shared/infrastructure/logging"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
	"go.uber.org/zap"
)

type CreateNewSecretUseCase struct {
	repo   SecretMetadataRepository
	savers map[secretsDomain.SecretType]SecretDataSaver
	txm    sharedUseCases.TransactionManager
}

func NewCreateNewSecretUseCase(repo SecretMetadataRepository, savers map[secretsDomain.SecretType]SecretDataSaver, txm sharedUseCases.TransactionManager) *CreateNewSecretUseCase {
	return &CreateNewSecretUseCase{repo: repo, savers: savers, txm: txm}
}

func (uc *CreateNewSecretUseCase) Execute(
	ctx context.Context,
	uID domain.UserID,
	secretName string,
	secretType secretsDomain.SecretType,
	src io.ReadCloser,
) (secretsDomain.SecretsMetadata, error) {
	sm, err := secretsDomain.NewSecretMetadata(uID, secretType, secretName)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, err
	}

	err = uc.txm.WithinTransaction(ctx, func(txCtx context.Context) error {
		err := uc.repo.AddSecretMetadata(txCtx, sm)
		if err != nil {
			l.Logger.Error("ошибка сохранения метаинформации о секрете в БД", zap.Error(err))
			return err
		}
		saver, ok := uc.savers[secretType]
		if !ok {
			l.Logger.Error("неизвестный тип секрета", zap.String("secret type", secretType.ToString()))
			return errors.New("неизвестный тип секрета")
		}

		err = saver.SaveSecretData(txCtx, sm, src)
		if err != nil {
			l.Logger.Error("ошибка сохранения данных", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		return secretsDomain.SecretsMetadata{}, err
	}
	return sm, err
}
