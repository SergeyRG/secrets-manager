package app

import (
	"context"
	"errors"
	"io"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	FilesUseCases "github.com/SergeyRG/secrets-manager/server/internal/files/use-cases"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	SecretsUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type CreateNewBinarySecretVersionOrch struct {
	uploadFileUsecase       *FilesUseCases.UploadFileUsecase
	getFileUsecase          *FilesUseCases.GetFileUsecase
	addSecretVersionUseCase *SecretsUseCases.AddSecretVersionUseCase
}

func NewCreateNewBinarySecretVersionOrch(
	uploadFileUsecase *FilesUseCases.UploadFileUsecase,
	getFileUsecase *FilesUseCases.GetFileUsecase,
	addSecretVersionUseCase *SecretsUseCases.AddSecretVersionUseCase,
) *CreateNewBinarySecretVersionOrch {

	return &CreateNewBinarySecretVersionOrch{
		uploadFileUsecase:       uploadFileUsecase,
		getFileUsecase:          getFileUsecase,
		addSecretVersionUseCase: addSecretVersionUseCase,
	}
}

func (orch *CreateNewBinarySecretVersionOrch) Execute(
	ctx context.Context,
	uID domain.UserID,
	secretName string,
	src io.ReadCloser) (secretsDomain.SecretsMetadata, error) {

	fileID, err := orch.uploadFileUsecase.Execute(ctx, src)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, errors.New("ошибка загрузки файла")
	}
	localFile, err := orch.getFileUsecase.Execute(ctx, fileID)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, errors.New("ошибка загрузки файла")
	}

	sm, err := orch.addSecretVersionUseCase.Execute(ctx, uID, secretName, localFile)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, errors.New("Ошибка создания бинарного секрета")
	}

	return sm, nil
}
