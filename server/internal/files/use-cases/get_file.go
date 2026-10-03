package usecases

import (
	"context"
	"fmt"
	"io"

	"github.com/SergeyRG/secrets-manager/server/internal/files/domain"
)

type GetFileUsecase struct {
	repo FilesRepository
}

func NewGetFileUsecase(repo FilesRepository) *GetFileUsecase {
	return &GetFileUsecase{repo: repo}
}

func (uc *GetFileUsecase) Execute(ctx context.Context, fID domain.FileID) (io.ReadCloser, error) {
	localFile, err := uc.repo.GetFile(ctx, fID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения файла: %w", err)
	}
	return localFile, nil
}
