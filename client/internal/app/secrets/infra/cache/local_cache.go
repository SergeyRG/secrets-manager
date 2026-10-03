package cache

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/fs"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/mem"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	"github.com/SergeyRG/secrets-manager/internal/shared/infrastructure/utils"
)

type metadataDTO struct {
	SecretName string            `json:"secret_name"`
	SecretType domain.SecretType `json:"secret_type"`
	Version    int64             `json:"version"`
}

type LocalCache struct {
	path              string
	cacheMetadataRepo usecases.SecretMetadataRepository
	cacheDataRepo     usecases.SecretDataRepository
}

func NewLocalCache(
	baseDir string,
	login string,
	salt []byte,
) *LocalCache {
	cacheID := calcLoginHash(login, salt)
	cacheDir := filepath.Join(baseDir, cacheID, "secrets")
	storage := make(map[string]map[int]domain.SecretsMetadata, 0)
	latest := make(map[string]int, 0)
	sortedSecretsName := make([]string, 0)
	metsRepo := mem.NewInMemoryMetadataRepository(storage, latest, sortedSecretsName)

	return &LocalCache{path: cacheDir, cacheMetadataRepo: metsRepo, cacheDataRepo: nil}
}

func (lc *LocalCache) Init(ctx context.Context) error {
	if lc.doesExists() {
		return lc.openLocalCache(ctx)
	}

	return lc.createNew()
}

func (lc *LocalCache) GetDataRepo() usecases.SecretDataRepository {
	return lc.cacheDataRepo
}

func (lc *LocalCache) GetMetadataRepo() usecases.SecretMetadataRepository {
	return lc.cacheMetadataRepo
}

func (lc *LocalCache) doesExists() bool {
	return utils.DirExists(lc.path)
}

func (lc *LocalCache) createNew() error {
	err := os.MkdirAll(lc.path, 0755)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(lc.path, "m"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	f.Close()
	dataRepo := fs.NewFSCacheDataRepository(lc.path)
	lc.cacheDataRepo = dataRepo
	return nil
}

func (lc *LocalCache) openLocalCache(ctx context.Context) error {
	metadata, err := os.ReadFile(filepath.Join(lc.path, "m"))
	if err != nil {
		return errors.New("ошибка чтения перечня метаданных из локального кэша")
	}
	dec := json.NewDecoder(bytes.NewReader(metadata))
	for dec.More() {
		mDTO := metadataDTO{}
		if err := dec.Decode(&mDTO); err != nil {
			return fmt.Errorf("ошибка парсинга json: %w", err)
		}
		sm := domain.SecretsMetadata{
			SecretName: mDTO.SecretName,
			SecretType: mDTO.SecretType,
			Version:    mDTO.Version,
		}
		err := lc.cacheMetadataRepo.AddSecretMetadata(ctx, sm)
		if err != nil {
			return fmt.Errorf("ошибка добаления метаданных в репозиторий: %w", err)
		}
	}
	lc.cacheDataRepo = fs.NewFSCacheDataRepository(lc.path)
	return nil
}

func (lc *LocalCache) SaveLocalCache(
	ctx context.Context,
) error {
	file, err := os.OpenFile(filepath.Join(lc.path, "m"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла локального кэша для записи: %v", err)
	}
	defer file.Close()
	page := 1
	perPage := 100
	newlineBytes := []byte("\n")
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		pg, err := lc.cacheMetadataRepo.GetUserSecretsMetadataPage(ctx, page, perPage)
		if err != nil {
			return nil
		}
		hasDataOnPage := false

		for _, meta := range pg {
			hasDataOnPage = true
			metaDTO := metadataDTO{
				SecretName: meta.SecretName,
				SecretType: meta.SecretType,
				Version:    meta.Version,
			}

			jsonData, err := json.Marshal(metaDTO)
			if err != nil {
				return fmt.Errorf("ошибка кодирования метаданных: %w", err)
			}

			if _, err := file.Write(jsonData); err != nil {
				return fmt.Errorf("ошибка сохранения данных: %w", err)
			}
			if _, err := file.Write(newlineBytes); err != nil {
				return fmt.Errorf("ошибка сохранения данных: %w", err)
			}
		}

		if !hasDataOnPage {
			break
		}

		page++
	}

	return nil
}

func calcLoginHash(login string, salt []byte) string {
	cleanLogin := strings.ToLower(strings.TrimSpace(login))
	h := hmac.New(sha256.New, salt)
	h.Write([]byte(cleanLogin))
	return hex.EncodeToString(h.Sum(nil))
}
