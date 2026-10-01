package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	authCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/cli"
	authclient "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/resty_passwd_auth_client"
	tokenstorage "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/token_storage"
	authUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases"
	orch "github.com/SergeyRG/secrets-manager/client/internal/app/orchestrators"
	registreCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/registre/infra/cli"
	registreRestyInfra "github.com/SergeyRG/secrets-manager/client/internal/app/registre/infra/resty"
	registreUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/registre/usecases"
	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	SecretsCacheInfra "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/cache"
	SecretsCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/cli"
	SecretsRestyInfra "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/resty"
	secretsUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	securityDomain "github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	securityCacheInfra "github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/cache"
	securityCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/cli"
	securityCryptoInfra "github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/crypto"
	securityRestyInfra "github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/resty"
	securityUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"github.com/SergeyRG/secrets-manager/client/internal/config"
	"github.com/SergeyRG/secrets-manager/internal/shared/infrastructure/logging"
	"resty.dev/v3"
)

func main() {
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := config.InitConfig()
	if err != nil {
		fmt.Printf("ошибка конфигурации приложения: %v\n", err)
		os.Exit(1)
	}

	baseLogger, err := logging.Initialize(config.LoggingLevel)
	if err != nil {
		fmt.Printf("ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	l := baseLogger.Named("main")

	restyClient := resty.New().SetBaseURL(config.ServerBaseUrl)
	consolePromter := sharedCli.NewConsolePrompter()

	if config.RegistreMode {
		dp := registreCliInfra.NewConsoleDataProvider(consolePromter)
		r := registreRestyInfra.NewRestyRegistreProvider(restyClient, config.RegistreRelativeURL)
		ruc := registreUsecases.NewRegistreUserUseCase(dp, r)

		err := ruc.Execute(context.Background())
		if err != nil {
			consolePromter.Send("Ошибка регистрации.\n")
			os.Exit(1)
		}
		consolePromter.Send("Вы успешно зарегистрированный, можете войти в систему.\n")
	}
	sessionType := sharedUsecases.SessionTypeRemote

	LoginPasswdProvider := authCliInfra.NewConsolePasswdCredsProvider(consolePromter)
	rac := authclient.NewRestyPasswdAuthClient(restyClient, LoginPasswdProvider)

	loginUC := authUsecases.NewLoginUseCase(rac)
	ts := tokenstorage.NewJWTTokenStorage(make([]byte, 0))

	login, err := loginUC.Execute(sigCtx, ts)
	if err != nil {
		if errors.Is(err, authUsecases.ErrServerUnavailable) {
			consolePromter.Send("Сервер не доступен. Открывается локальная сессия\n")
			sessionType = sharedUsecases.SessionTypeLocal
		} else {
			consolePromter.Send("ошибка аутентификации.\n")
			os.Exit(1)
		}
	}

	LocalKeyCache := securityCacheInfra.NewLocalCache(
		config.LocalCacheDir,
		login,
		config.CacheIDSalt,
	)

	if sessionType == sharedUsecases.SessionTypeLocal && !LocalKeyCache.DoesExists() {
		consolePromter.Send("Локальный кэш не существует. Дальнейшая работа не возможна.")
		os.Exit(0)
	}
	err = LocalKeyCache.Init()
	if err != nil {
		consolePromter.Send(fmt.Sprintf("ошибка инициализации локального кэша: %v", err))
		os.Exit(1)
	}

	RemoteKeyRepo := securityRestyInfra.NewRestyEncryptedKeyRepo(restyClient, config.SecurityKeyRelaitiveURL)
	getKeyUC := securityUsecases.NewGetUserKeyUseCase(
		RemoteKeyRepo,
		LocalKeyCache.GetRepo(),
		sessionType,
	)

	ks := securityDomain.NewKeyStorage(
		securityCryptoInfra.DecryptDataWithPassword,
		securityCryptoInfra.EncryptDataWithPassword,
		securityCryptoInfra.EncryptStream,
		securityCryptoInfra.DecryptStream,
	)

	pp := securityCliInfra.NewCliPasswordProvider(consolePromter)

	createKeyUC := securityUsecases.NewCreateKeyUseCase(RemoteKeyRepo, LocalKeyCache.GetRepo(), sessionType)

	err = getKeyUC.Execute(sigCtx, ks, pp)
	if err != nil {
		if errors.Is(err, securityUsecases.ErrKeyNotExist) {
			err = createKeyUC.Execute(sigCtx, pp, ks)
			if err != nil {
				consolePromter.Send("ошибка создания ключа.\n")
				os.Exit(1)
			}
		} else {
			consolePromter.Send("ошибка получения ключа.\n")
			os.Exit(1)
		}
	}

	l.Info("Успешная авторизация пользователя")

	secretsMetadataRepo := SecretsRestyInfra.NewRestySecretsRepo(
		restyClient,
		config.MetadataRelativeURL,
		config.MetadataListRelativeURL,
	)
	secretsDataRepo := SecretsRestyInfra.NewRestySecretsDataRepo(
		restyClient,
		config.TextDataRelativeURL,
		config.BlobDataRelativeURL,
	)

	localSecretsCache := SecretsCacheInfra.NewLocalCache(config.LocalCacheDir, login, config.CacheIDSalt)
	localSecretsCache.Init(sigCtx)

	registry := make(map[string]sharedCli.CommandRegistryEntry, 0)

	// команда list //

	listCommandInfo := sharedCli.CommandInfo{
		Name:        "list",
		Description: "вывести список всех секретов",
	}
	getUserSecretsUC := secretsUsecases.NewGetSecretsListUseCase(
		secretsMetadataRepo,
		localSecretsCache.GetMetadataRepo(),
		sessionType,
	)
	getUserSecretsListHandler := SecretsCliInfra.NewGetSecretListCommandHandler(
		getUserSecretsUC,
		config.PerPage)
	registry["list"] = sharedCli.CommandRegistryEntry{
		CommandInfo: listCommandInfo,
		Handler:     getUserSecretsListHandler,
	}

	// команда get //

	getCommandInfo := sharedCli.CommandInfo{
		Name:        "get",
		Description: `получить данные секрета. Использование: get <ИМЯ СЕКРЕТА> [<Версия>]. Например, get test_secret 2`,
	}

	getMetadataUC := secretsUsecases.NewGetSecretMetadataUseCase(
		localSecretsCache.GetMetadataRepo(),
		secretsMetadataRepo,
		sessionType,
	)
	getDataUC := secretsUsecases.NewGetSecretDataUseCase(
		secretsDataRepo,
		localSecretsCache.GetDataRepo(),
		localSecretsCache.GetMetadataRepo(),
		sessionType,
	)
	getDataOrch := orch.NewGetSecretDataByNameOrch(getMetadataUC, getDataUC, ks)

	views := make(map[secretsDomain.SecretType]SecretsCliInfra.View, 0)
	views[secretsDomain.SecretTypeAuthData] = &SecretsCliInfra.AuthDataSecretViewer{}
	views[secretsDomain.SecretTypeBankCard] = &SecretsCliInfra.BankDataSecretViewer{}
	views[secretsDomain.SecretTypeFreeText] = &SecretsCliInfra.FreeTextSecretViewer{}
	views[secretsDomain.SecretTypeBinary] = &SecretsCliInfra.BlobSecretViewer{DataDir: config.BinarySecretsLoadDir}

	getDataHandler := SecretsCliInfra.NewGetSecretDataCommandHandler(getDataOrch, views)
	registry["get"] = sharedCli.CommandRegistryEntry{
		CommandInfo: getCommandInfo,
		Handler:     getDataHandler,
	}

	// команда create //

	createCommandInfo := sharedCli.CommandInfo{
		Name:        "create",
		Description: `создать новый секрет. Использование: create <ИМЯ СЕКРЕТА>`,
	}

	getters := make(map[secretsDomain.SecretType]SecretsCliInfra.SecretDataGetter)
	getters[secretsDomain.SecretTypeBinary] = &SecretsCliInfra.BlobSecretGetter{MaxSizeBytes: config.MaxSizeBytes}
	getters[secretsDomain.SecretTypeFreeText] = &SecretsCliInfra.FreeTextSecretGetter{}
	getters[secretsDomain.SecretTypeAuthData] = &SecretsCliInfra.AuthDataSecretGetter{}
	getters[secretsDomain.SecretTypeBankCard] = &SecretsCliInfra.BankDataSecretGetter{}

	createSecretUc := secretsUsecases.NewCreateSecretUseCase(secretsDataRepo, sessionType)
	createSecretOrch := orch.NewCreateSecretOrch(createSecretUc, ks)
	createCommandHandler := SecretsCliInfra.NewCreateSecretDataCommandHandler(createSecretOrch, getters)
	registry["create"] = sharedCli.CommandRegistryEntry{
		CommandInfo: createCommandInfo,
		Handler:     createCommandHandler,
	}

	commandRouter := sharedCli.NewCommandRouter(registry)
	repl := sharedCli.NewREPL(commandRouter, consolePromter)

	err = repl.Start(sigCtx)
	if err != nil {
		consolePromter.Send(fmt.Sprintf("приложение завершено с ошибкой: %v", err))
	}
	defer func() {
		localSecretsCache.SaveLocalCache(context.Background())
	}()
}
