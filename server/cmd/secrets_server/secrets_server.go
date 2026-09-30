package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/SergeyRG/secrets-manager/internal/shared/infrastructure/logging"
	"github.com/SergeyRG/secrets-manager/server/internal/app"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/crypto"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/db/psql"
	authHandlers "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/http/handlers"
	authMiddleware "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/http/middleware"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases"
	"github.com/SergeyRG/secrets-manager/server/internal/config"
	"github.com/SergeyRG/secrets-manager/server/internal/files/infrastructure/db/filesystem"
	filesPsql "github.com/SergeyRG/secrets-manager/server/internal/files/infrastructure/db/psql"
	filesUseCases "github.com/SergeyRG/secrets-manager/server/internal/files/use-cases"
	"github.com/SergeyRG/secrets-manager/server/internal/migrations"
	secretsDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	secretsFS "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/db/filesystem"
	secretsPSQL "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/db/psql"
	secretsHandlers "github.com/SergeyRG/secrets-manager/server/internal/secrets/infrastructure/http"
	secretsUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
	secretsDataReceivers "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/receivers"
	secretsDataSavers "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases/savers"
	securityPSQL "github.com/SergeyRG/secrets-manager/server/internal/security/infrastructure/db/psql"
	securityHandlers "github.com/SergeyRG/secrets-manager/server/internal/security/infrastructure/http"
	sharedPsql "github.com/SergeyRG/secrets-manager/server/internal/shared/infrastructure/psql"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	//Чтение конфигурационного файла
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Printf("ошибка конфигурации: %v", err)
		return
	}

	//Инициализация логера
	baseLogger, err := logging.Initialize(cfg.LoggingLevel)
	if err != nil {
		log.Fatal("ошибка инициализации системы логирования", zap.Error(err))
		return
	}
	l := baseLogger.Named("main")

	//Подключение к БД и проверка соединения
	db, err := sharedPsql.CreateAndCheckPSQLCon(cfg.DBDSN)
	if err != nil {
		l.Fatal("ошибка соединения с БД", zap.Error(err))
		return
	}

	//Запуск миграций БД
	err = migrations.RunMigrations(db)
	if err != nil {
		l.Fatal("ошибка обновления схемы БД", zap.Error(err))
		return
	}

	//Инициализация менеджера транзакций. Используется в репозиториях и юз кейсах
	txm := sharedPsql.NewTxManager(db)

	//Инициализация остальных объектов приложения и настройка роутера
	r, err := InitApp(cfg, txm, baseLogger)
	if err != nil {
		l.Fatal("ошибка инициализации приложения", zap.Error(err))
		return
	}

	err = http.ListenAndServe(cfg.Address, r)
	if err != nil {
		l.Error("Сервер завершил работы с ошибкой: %v", zap.Error(err))
	}
}

func InitApp(cfg config.Config, txm *sharedPsql.PSQLTxManager, bl *zap.Logger) (*chi.Mux, error) {
	r := chi.NewRouter()
	jwtm := crypto.NewJWTManager([]byte(cfg.SecretKey))

	userRepo := psql.NewPSQLUserRepo(txm)
	authUserUseCase := usecases.NewAuthUserUseCase(userRepo, crypto.Hasher{})
	registreUseCase := usecases.NewRegisterUserUseCase(userRepo, crypto.Hasher{})

	loginHandler := authHandlers.NewLoginHandler(authUserUseCase, jwtm)
	registreHandler := authHandlers.NewRegistreHandler(registreUseCase, jwtm)
	authMiddleware := authMiddleware.NewAuthMiddleware(jwtm)

	r.Post("/api/user/register", registreHandler.Handle)
	r.Post("/api/user/login", loginHandler.Handle)

	filesRepo := filesystem.NewFilesystemFilesRepo(cfg.BaseUploadFilesDirectory)
	filesStateRepo := filesPsql.NewPSQLFileStateRepository(txm)

	uploadFileUsecase := filesUseCases.NewUploadFileUsecase(filesRepo, filesStateRepo)
	getFileUseCase := filesUseCases.NewGetFileUsecase(filesRepo)

	secretsMetaRepo := secretsPSQL.NewSecretsMetadataRepo(txm)
	secretsTextRepo := secretsPSQL.NewSecretsTextdataRepo(txm)
	secretsBlobRepo := secretsFS.NewSecretsBlobDataRepo(cfg.BaseBlobRepoFilesDirectory)

	savers := make(map[secretsDomain.SecretType]secretsUseCases.SecretDataSaver)

	textSecretSaver := secretsDataSavers.NewTextSecretSaver(secretsTextRepo)
	blobSecretSaver := secretsDataSavers.NewBlobSecretSaver(secretsBlobRepo)

	savers[secretsDomain.SecretTypeFreeText] = textSecretSaver
	savers[secretsDomain.SecretTypeBankCard] = textSecretSaver
	savers[secretsDomain.SecretTypeAuthData] = textSecretSaver
	savers[secretsDomain.SecretTypeBinary] = blobSecretSaver

	createNewSecretUC := secretsUseCases.NewCreateNewSecretUseCase(secretsMetaRepo, savers, txm, bl.Named("create_secret"))

	CreateNewBinarySecretOrch := app.NewCreateNewSecretOrch(uploadFileUsecase, getFileUseCase, createNewSecretUC)

	CreateNewBinarySecretHandler := secretsHandlers.NewCreateNewBinarySecretHandler(CreateNewBinarySecretOrch)
	r.With(authMiddleware).Post("/api/secrets/binary", CreateNewBinarySecretHandler.Handle)

	createNewSecretVersionUC := secretsUseCases.NewAddSecretVersionUseCase(secretsMetaRepo, savers, txm, bl.Named("create_secret_version"))
	createNewBinarySecretVersionOrch := app.NewCreateNewBinarySecretVersionOrch(uploadFileUsecase, getFileUseCase, createNewSecretVersionUC)
	createNewBinarySecretVersionHandler := secretsHandlers.NewCreateNewBinarySecretVersionHandler(createNewBinarySecretVersionOrch)
	r.With(authMiddleware).Patch("/api/secrets/binary", createNewBinarySecretVersionHandler.Handle)

	CreateNewTextSecretHandler := secretsHandlers.NewCreateNewTextSecretHandler(createNewSecretUC)
	r.With(authMiddleware).Post("/api/secrets/text", CreateNewTextSecretHandler.Handle)

	CreateNewTextSecretVersionHandler := secretsHandlers.NewCreateNewTextSecretVersionHandler(createNewSecretVersionUC)
	r.With(authMiddleware).Patch("/api/secrets/text", CreateNewTextSecretVersionHandler.Handle)

	receivers := make(map[secretsDomain.SecretType]secretsUseCases.SecretDataReceiver)

	textSecretReceiver := secretsDataReceivers.NewTextSecretReceiver(secretsTextRepo)
	blobSecretReceiver := secretsDataReceivers.NewBlobSecretReceiver(secretsBlobRepo)
	receivers[secretsDomain.SecretTypeFreeText] = textSecretReceiver
	receivers[secretsDomain.SecretTypeAuthData] = textSecretReceiver
	receivers[secretsDomain.SecretTypeBankCard] = textSecretReceiver
	receivers[secretsDomain.SecretTypeBinary] = blobSecretReceiver

	getSecretVersionUC := secretsUseCases.NewGetSecretVersionUseCase(secretsMetaRepo, receivers, txm, bl.Named("get_secret"))
	getTextSecretVersionHandler := secretsHandlers.NewGetTextSecretVersionHandler(getSecretVersionUC)
	r.With(authMiddleware).Get("/api/secrets/text", getTextSecretVersionHandler.Handle)

	getBlobSecretVersionHandler := secretsHandlers.NewGetBlobSecretVersionHandler(getSecretVersionUC)
	r.With(authMiddleware).Get("/api/secrets/binary", getBlobSecretVersionHandler.Handle)

	GetUserSecretMetadataPageUC := secretsUseCases.NewGetUserSecretMetadataPageUseCase(secretsMetaRepo, txm)
	getUserSecretMetadataPageHandler := secretsHandlers.NewGetUserSecretMetadataPageHandler(GetUserSecretMetadataPageUC)
	r.With(authMiddleware).Get("/api/secrets/list", getUserSecretMetadataPageHandler.Handle)

	securityRepo := securityPSQL.NewPSQLEncryptedKeyRepo(txm)
	getUserEncryptedKeyHandler := securityHandlers.NewGetUserEncryptedKeyHandler(securityRepo)
	setUserEncryptedKeyHandler := securityHandlers.NewSetUserEncryptedKeyHandler(securityRepo)
	r.With(authMiddleware).Get("/api/security/key", getUserEncryptedKeyHandler.Handle)
	r.With(authMiddleware).Post("/api/security/key", setUserEncryptedKeyHandler.Handle)

	getUserSecretMetadataUC := secretsUseCases.NewGetUserSecretMetadataUseCase(secretsMetaRepo)
	getUserSecretMetadataHandler := secretsHandlers.NewGetUserSecretMetadataHandler(getUserSecretMetadataUC)
	r.With(authMiddleware).Get("/api/secrets/metadata", getUserSecretMetadataHandler.Handle)

	return r, nil
}
