package config

import (
	"errors"
	"flag"
	"os"
)

type Config struct {
	DBDSN                      string
	LoggingLevel               string
	SecretKey                  string
	Address                    string
	BaseUploadFilesDirectory   string
	BaseBlobRepoFilesDirectory string
}

func NewConfig() (Config, error) {
	cfg := Config{
		DBDSN:                      "postgres://test:test@localhost:5432/secrets?sslmode=disable",
		LoggingLevel:               "debug",
		SecretKey:                  "",
		Address:                    ":8080",
		BaseUploadFilesDirectory:   "",
		BaseBlobRepoFilesDirectory: "",
	}

	t := flag.String("t", "", "путь к временной папке для загрузки бинарных файлов")
	b := flag.String("b", "", "путь к директории хранения бинарных файлов")
	d := flag.String("d", "", "DBDSN")

	flag.Parse()

	secretKey, ok := os.LookupEnv("SECRET_KEY")
	if !ok {
		return Config{}, errors.New("Переменная среды с секретным ключем не задана")
	}

	cfg.SecretKey = secretKey

	if *t == "" || *b == "" {
		return Config{}, errors.New("не заданы пути хранения бинарных данных")
	}

	cfg.BaseUploadFilesDirectory = *t
	cfg.BaseBlobRepoFilesDirectory = *b

	if *d != "" {
		cfg.DBDSN = *d
	}
	return cfg, nil
}
