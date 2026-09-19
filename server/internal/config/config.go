package config

type Config struct {
	DBDSN                      string
	LoggingLevel               string
	SecretKey                  string
	Address                    string
	BaseUploadFilesDirectory   string
	BaseBlobRepoFilesDirectory string
}

func NewConfig() Config {
	return Config{
		DBDSN:                      "postgres://test:test@localhost:5432/secrets?sslmode=disable",
		LoggingLevel:               "debug",
		SecretKey:                  "test",
		Address:                    ":8080",
		BaseUploadFilesDirectory:   "D:\\temp",
		BaseBlobRepoFilesDirectory: "D:\\temp\\repo",
	}
}
