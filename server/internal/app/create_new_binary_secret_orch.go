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

type CreateNewBinarySecretOrch struct {
	uploadFileUsecase      *FilesUseCases.UploadFileUsecase
	getFileUsecase         *FilesUseCases.GetFileUsecase
	createNewSecretUseCase *SecretsUseCases.CreateNewSecretUseCase
}

func NewCreateNewSecretOrch(
	uploadFileUsecase *FilesUseCases.UploadFileUsecase,
	getFileUsecase *FilesUseCases.GetFileUsecase,
	createNewSecretUseCase *SecretsUseCases.CreateNewSecretUseCase,
) *CreateNewBinarySecretOrch {

	return &CreateNewBinarySecretOrch{
		uploadFileUsecase:      uploadFileUsecase,
		getFileUsecase:         getFileUsecase,
		createNewSecretUseCase: createNewSecretUseCase,
	}
}

func (orch *CreateNewBinarySecretOrch) Execute(
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
		return secretsDomain.SecretsMetadata{}, errors.New("ошибка открытия файла")
	}

	sm, err := orch.createNewSecretUseCase.Execute(ctx, uID, secretName, secretsDomain.SecretTypeBinary, localFile)
	if err != nil {
		return secretsDomain.SecretsMetadata{}, errors.New("Ошибка создания бинарного секрета")
	}

	return sm, nil
}
