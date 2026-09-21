package config

import (
	"errors"
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
	TextDataRelativeURL     string
	BlobDataRelativeURL     string
	MetadataRelativeURL     string
	SecurityKeyRelaitiveURL string
	RegistreRelativeURL     string
	RegistreMode            bool
}

func newConfig() Config {
	return Config{
		ServerBaseUrl:           "http://localhost:8080",
		LoggingLevel:            "debug",
		PerPage:                 10,
		BinarySecretsLoadDir:    "D:\\Games",
		TextDataRelativeURL:     "api/secrets/text",
		BlobDataRelativeURL:     "api/secrets/binary",
		MetadataRelativeURL:     "api/secrets/metadata",
		SecurityKeyRelaitiveURL: "api/security/key",
		RegistreRelativeURL:     "api/user/register",
		RegistreMode:            false,
	}
}

func InitConfig() (Config, error) {
	cfg := newConfig()

	r := flag.Bool("r", false, "Запустить в режиме регистрации")
	d := flag.String("d", "", "путь к директории сохранения бинарных файлов")

	flag.Parse()

	if *d == "" {
		execPath, err := os.Executable()
		if err != nil {
			fmt.Printf("Ошибка получения пути к исполняемому файлу: %v\n", err)
			return Config{}, err
		}
		fmt.Println("Путь к файлу:", execPath)
		cfg.BinarySecretsLoadDir = filepath.Dir(execPath)
	} else {
		if !dirExists(*d) {
			return Config{}, fmt.Errorf("ошибка проверки доступа к директории %v", *d)
		}
		cfg.BinarySecretsLoadDir = *d
	}
	cfg.RegistreMode = *r
	return cfg, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir()
	}

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return false
}
