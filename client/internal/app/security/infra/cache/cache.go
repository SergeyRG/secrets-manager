package cache

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/fs"
	"github.com/SergeyRG/secrets-manager/internal/shared/infrastructure/utils"
)

type LocalCache struct {
	path         string
	cacheKeyRepo domain.EncryptedKeyRepository
}

func NewLocalCache(
	baseDir string,
	login string,
	salt []byte,
) *LocalCache {
	cacheID := calcLoginHash(login, salt)
	cacheFile := filepath.Join(baseDir, cacheID, "sec", "k")
	return &LocalCache{path: cacheFile, cacheKeyRepo: nil}
}

func (lc *LocalCache) Init() error {
	if lc.DoesExists() {
		return lc.openLocalCache()
	}

	return lc.createNew()
}

func (lc *LocalCache) GetRepo() domain.EncryptedKeyRepository {
	return lc.cacheKeyRepo
}

func (lc *LocalCache) DoesExists() bool {
	return utils.FileExists(lc.path)
}

func (lc *LocalCache) createNew() error {
	err := os.MkdirAll(filepath.Dir(lc.path), 0700)
	if err != nil {
		return err
	}
	file, err := os.Create(lc.path)
	if err != nil {
		return err
	}
	defer file.Close()
	lc.cacheKeyRepo = fs.NewFSEncryptedKeyRepo(lc.path)
	return nil
}

func (lc *LocalCache) openLocalCache() error {
	lc.cacheKeyRepo = fs.NewFSEncryptedKeyRepo(lc.path)
	return nil
}

func calcLoginHash(login string, salt []byte) string {
	cleanLogin := strings.ToLower(strings.TrimSpace(login))
	h := hmac.New(sha256.New, salt)
	h.Write([]byte(cleanLogin))
	return hex.EncodeToString(h.Sum(nil))
}
