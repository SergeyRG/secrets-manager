package usecases

import (
	"context"
	"fmt"
	"io"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	sharedUseCases "github.com/SergeyRG/secrets-manager/server/internal/shared/use-cases"
	"go.uber.org/zap"
)

type AddSecretVersionUseCase struct {
	repo   SecretMetadataRepository
	savers map[secretsDomain.SecretType]SecretDataSaver
	txm    sharedUseCases.TransactionManager
	l      *zap.Logger
}

func NewAddSecretVersionUseCase(
	repo SecretMetadataRepository,
	savers map[secretsDomain.SecretType]SecretDataSaver,
	txm sharedUseCases.TransactionManager,
	l *zap.Logger,
) *AddSecretVersionUseCase {
	return &AddSecretVersionUseCase{repo: repo, txm: txm, savers: savers, l: l}
}

func (uc *AddSecretVersionUseCase) Execute(
	ctx context.Context,
	uID domain.UserID,
	sName string,
	src io.ReadCloser,
) (secretsDomain.SecretsMetadata, error) {

	var sm secretsDomain.SecretsMetadata

	err := uc.txm.WithinTransaction(ctx, func(txCtx context.Context) error {
		curSm, err := uc.repo.GetUserSecretMetadataByName(txCtx, uID, sName, 0)
		if err != nil {
			uc.l.Error("ошибка получения метаданных секрета из БД", zap.Error(err))
			return err
		}
		newSm, err := secretsDomain.NewSecretVersionMetadata(curSm)
		if err != nil {
			uc.l.Error("ошибка создания метаданных новой версии секрета", zap.Error(err))
			return err
		}

		err = uc.repo.AddSecretMetadata(txCtx, newSm)
		if err != nil {
			uc.l.Error("ошибка сохранения метаинформации о секрете в БД", zap.Error(err))
			return err
		}

		saver, ok := uc.savers[newSm.SecretType]
		if !ok {
			uc.l.Error("неизвестный тип секрета", zap.String("secret type", sm.SecretType.ToString()))
			return fmt.Errorf("неизвестный тип секрета: %v", newSm.SecretType)
		}
		err = saver.SaveSecretData(txCtx, newSm, src)
		if err != nil {
			uc.l.Error("ошибка сохранения данных", zap.Error(err))
			return err
		}
		sm = newSm

		return nil
	})

	return sm, err
}
