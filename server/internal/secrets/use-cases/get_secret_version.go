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

type GetSecretVersionUseCase struct {
	repo      SecretMetadataRepository
	receivers map[secretsDomain.SecretType]SecretDataReceiver
	txm       sharedUseCases.TransactionManager
	l         *zap.Logger
}

func NewGetSecretVersionUseCase(
	repo SecretMetadataRepository,
	receivers map[secretsDomain.SecretType]SecretDataReceiver,
	txm sharedUseCases.TransactionManager,
	l *zap.Logger,

) *GetSecretVersionUseCase {
	return &GetSecretVersionUseCase{
		repo:      repo,
		txm:       txm,
		receivers: receivers,
		l:         l,
	}
}

func (uc *GetSecretVersionUseCase) Execute(
	ctx context.Context,
	uID domain.UserID,
	sName string,
	version int,
) (io.ReadCloser, error) {

	sm, err := uc.repo.GetUserSecretMetadataByName(ctx, uID, sName, version)
	if err != nil {
		uc.l.Error("ошибка получения метаданных секрета из БД", zap.Error(err))
		return nil, err
	}

	receiver, ok := uc.receivers[sm.SecretType]
	if !ok {
		uc.l.Error("неизвестный тип секрета", zap.String("secret type", sm.SecretType.ToString()))
		return nil, fmt.Errorf("неизвестный тип секрета: %v", sm.SecretType)
	}
	data, err := receiver.ReceiveSecretData(ctx, sm)
	if err != nil {
		uc.l.Error("ошибка чтения данных", zap.Error(err))
		return nil, err
	}

	return data, err
}
