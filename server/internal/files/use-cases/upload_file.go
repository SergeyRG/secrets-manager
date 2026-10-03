package usecases

import (
	"context"
	"fmt"
	"io"

	"github.com/SergeyRG/secrets-manager/server/internal/files/domain"
)

type UploadFileUsecase struct {
	repo      FilesRepository
	repoState FilesStateRepository
}

func NewUploadFileUsecase(repo FilesRepository, repoState FilesStateRepository) *UploadFileUsecase {
	return &UploadFileUsecase{repo: repo, repoState: repoState}
}

func (uc *UploadFileUsecase) Execute(ctx context.Context, src io.ReadCloser) (domain.FileID, error) {
	fileID := domain.NewFileID()
	err := uc.repoState.AddFileState(ctx, fileID, domain.FileStateUploading)
	if err != nil {
		return "", fmt.Errorf("ошибка записи состояния нового файла: %w", err)
	}

	err = uc.repo.AddFile(ctx, fileID, src)
	if err != nil {
		return "", fmt.Errorf("ошибка загрузки файла на сервер: %w", err)
	}

	err = uc.repoState.AddFileState(ctx, fileID, domain.FileStateReady)
	if err != nil {
		return "", fmt.Errorf("файл %v успешно загружен, ошибка изменения состояния: %w", fileID, err)
	}
	return fileID, nil
}
