package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SergeyRG/secrets-manager/server/internal/files/domain"
)

type FilesystemFilesRepo struct {
	BaseDirectoryPath string
}

func NewFilesystemFilesRepo(BaseDirectoryPath string) *FilesystemFilesRepo {
	return &FilesystemFilesRepo{BaseDirectoryPath: BaseDirectoryPath}
}

func (r *FilesystemFilesRepo) AddFile(ctx context.Context, fID domain.FileID, src io.ReadCloser) error {
	defer src.Close()

	path := filepath.Join(r.BaseDirectoryPath, string(fID))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, src)
	if err != nil {
		return fmt.Errorf("ошибка сохранения данных в файл: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия файла после записи: %w", err)
	}

	return nil
}

func (r *FilesystemFilesRepo) GetFile(ctx context.Context, fID domain.FileID) (io.ReadCloser, error) {
	path := filepath.Join(r.BaseDirectoryPath, string(fID))
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %w", err)
	}

	return file, nil
}
