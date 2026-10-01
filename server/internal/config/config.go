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
		DBDSN:                      "",
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

	if *t == "" || *b == "" || *d == "" {
		return Config{}, errors.New("не заданы обательные параметры запуска")
	}

	cfg.BaseUploadFilesDirectory = *t
	cfg.BaseBlobRepoFilesDirectory = *b
	cfg.DBDSN = *d

	return cfg, nil
}
