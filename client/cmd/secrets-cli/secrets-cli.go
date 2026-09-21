package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	authCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/cli"
	authclient "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/resty_passwd_auth_client"
	tokenstorage "github.com/SergeyRG/secrets-manager/client/internal/app/auth/infra/token_storage"
	authUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/auth/usecases"
	command_handlers "github.com/SergeyRG/secrets-manager/client/internal/app/command_handlers"
	orch "github.com/SergeyRG/secrets-manager/client/internal/app/orchestrators"
	registreCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/registre/infra/cli"
	registreRestyInfra "github.com/SergeyRG/secrets-manager/client/internal/app/registre/infra/resty"
	registreUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/registre/usecases"
	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	SecretsCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/cli"
	SecretsRestyInfra "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/resty"
	secretsUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	securityDomain "github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	securityCliInfra "github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/cli"
	securityCryptoInfra "github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/crypto"
	securityRestyInfra "github.com/SergeyRG/secrets-manager/client/internal/app/security/infra/resty"
	securityUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"

	"github.com/SergeyRG/secrets-manager/client/internal/config"
	"github.com/SergeyRG/secrets-manager/internal/shared/infrastructure/logging"
	"resty.dev/v3"
)

func main() {
	config, err := config.InitConfig()
	if err != nil {
		fmt.Printf("ошибка конфигурации приложения: %v\n", err)
		os.Exit(1)
	}

	logging.Initialize(config.LoggingLevel)
	l := logging.Logger

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

	LoginPasswdProvider := authCliInfra.NewConsolePasswdCredsProvider(consolePromter)
	rac := authclient.NewRestyPasswdAuthClient(restyClient, LoginPasswdProvider)

	loginUC := authUsecases.NewLoginUseCase(rac)

	ts := tokenstorage.NewJWTTokenStorage(make([]byte, 0))

	keyRepo := securityRestyInfra.NewRestyEncryptedKeyRepo(restyClient, config.SecurityKeyRelaitiveURL)
	getKeyUC := securityUsecases.NewGetUserKeyUseCase(keyRepo)
	ks := securityDomain.NewKeyStorage(
		securityCryptoInfra.DecryptKeyWithPassword,
		securityCryptoInfra.EncryptKeyWithPassword,
		securityCryptoInfra.EncryptStream,
		securityCryptoInfra.DecryptStream,
	)

	pp := securityCliInfra.NewCliPasswordProvider(consolePromter)

	createKeyUC := securityUsecases.NewCreateKeyUseCase(keyRepo)

	LoginOrch := orch.NewLoginOrchestrator(loginUC, createKeyUC, getKeyUC)
	err = LoginOrch.Execute(context.Background(), ts, ks, pp)
	if err != nil {
		consolePromter.Send(fmt.Sprintf("ошибка входа в приложение: %v\n", err))
		os.Exit(1)
	}

	l.Info("Успешная авторизация пользователя")

	registry := make(map[string]sharedCli.CommandRegistryEntry, 0)

	// команда list //

	listCommandInfo := sharedCli.CommandInfo{
		Name:        "list",
		Description: "вывести список всех секретов",
	}
	getUserSecretsListHandler := command_handlers.NewGetUserSecretsListHandler(
		restyClient,
		"/api/secrets/list",
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
	secretsMetadataRepo := SecretsRestyInfra.NewRestySecretsRepo(restyClient, config.MetadataRelativeURL)
	secretsDataRepo := SecretsRestyInfra.NewRestySecretsDataRepo(
		restyClient,
		config.TextDataRelativeURL,
		config.BlobDataRelativeURL,
	)

	getMetadataUC := secretsUsecases.NewGetSecretMetadataUseCase(secretsMetadataRepo)
	getDataUC := secretsUsecases.NewGetSecretDataUseCase(secretsDataRepo)
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
	getters[secretsDomain.SecretTypeBinary] = &SecretsCliInfra.BlobSecretGetter{}
	getters[secretsDomain.SecretTypeFreeText] = &SecretsCliInfra.FreeTextSecretGetter{}
	getters[secretsDomain.SecretTypeAuthData] = &SecretsCliInfra.AuthDataSecretGetter{}
	getters[secretsDomain.SecretTypeBankCard] = &SecretsCliInfra.BankDataSecretGetter{}

	createSecretUc := secretsUsecases.NewCreateSecretUseCase(secretsDataRepo)
	createSecretOrch := orch.NewCreateSecretOrch(createSecretUc, ks)
	createCommandHandler := SecretsCliInfra.NewCreateSecretDataCommandHandler(createSecretOrch, getters)
	registry["create"] = sharedCli.CommandRegistryEntry{
		CommandInfo: createCommandInfo,
		Handler:     createCommandHandler,
	}

	commandRouter := sharedCli.NewCommandRouter(registry)
	repl := sharedCli.NewREPL(commandRouter, consolePromter)

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repl.Start(sigCtx)
}
