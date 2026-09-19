package config

import "flag"

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

func InitConfig() Config {
	r := flag.Bool("r", false, "Запустить в режиме регистрации")
	flag.Parse()
	cfg := newConfig()
	cfg.RegistreMode = *r
	return cfg
}
