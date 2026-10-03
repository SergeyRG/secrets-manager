package config

import (
	"github.com/SergeyRG/secrets-manager/internal/shared/infrastructure/utils"

	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ServerBaseUrl           string
	LoggingLevel            string
	PerPage                 int
	BinarySecretsLoadDir    string
	LocalCacheDir           string
	CacheIDSalt             []byte
	TextDataRelativeURL     string
	BlobDataRelativeURL     string
	MetadataRelativeURL     string
	MetadataListRelativeURL string
	SecurityKeyRelaitiveURL string
	RegistreRelativeURL     string
	RegistreMode            bool
	MaxSizeBytes            int64
}

func newConfig() Config {
	return Config{
		ServerBaseUrl:           "http://localhost:8080",
		LoggingLevel:            "debug",
		PerPage:                 10,
		BinarySecretsLoadDir:    "",
		TextDataRelativeURL:     "api/secrets/text",
		BlobDataRelativeURL:     "api/secrets/binary",
		MetadataRelativeURL:     "api/secrets/metadata",
		MetadataListRelativeURL: "api/secrets/list",
		SecurityKeyRelaitiveURL: "api/security/key",
		RegistreRelativeURL:     "api/user/register",
		RegistreMode:            false,
		MaxSizeBytes:            1024 * 1024 * 100, //100mb
	}
}

func InitConfig() (Config, error) {
	cfg := newConfig()

	r := flag.Bool("r", false, "Запустить в режиме регистрации")
	d := flag.String("d", "", "путь к директории сохранения бинарных файлов")

	flag.Parse()

	execPath, err := os.Executable()
	execDir := filepath.Dir(execPath)
	if err != nil {
		fmt.Printf("Ошибка получения пути к исполняемому файлу: %v\n", err)
		return Config{}, err
	}
	cfg.LocalCacheDir = filepath.Join(execDir, "cache")

	if *d == "" {
		cfg.BinarySecretsLoadDir = filepath.Dir(execPath)
	} else {
		if !utils.DirExists(*d) {
			return Config{}, fmt.Errorf("ошибка проверки доступа к директории %v", *d)
		}
		cfg.BinarySecretsLoadDir = *d
	}
	cfg.RegistreMode = *r
	cfg.CacheIDSalt = []byte{
		0x0a, 0x7f, 0x3d, 0x81, 0xb9, 0xc2, 0x4e, 0x5a,
		0x6f, 0x8b, 0x90, 0x1c, 0x2d, 0x3e, 0x4f, 0x5a,
		0x6b, 0x7c, 0x8d, 0x9e, 0x0f, 0x1a, 0x2b, 0x3c,
		0x4d, 0x5e, 0x6f, 0x7a, 0x8b, 0x9c, 0x0d, 0x1e,
	}
	return cfg, nil
}
