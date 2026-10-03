package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type FreeTextSecretViewer struct {
	SecretFreeText
}

func (v *FreeTextSecretViewer) ProvideDataToUser(
	smt domain.SecretsMetadata,
	data io.ReadCloser,
	promter sharedCli.Prompter) error {
	defer data.Close()
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return errors.New("ошибка чтения секрета")
	}
	err = json.Unmarshal(dataBytes, v)
	if err != nil {
		return errors.New("парсинга данных секрета")
	}
	err = promter.Send(fmt.Sprintf("данные: %s\n", v.Data))
	if err != nil {
		return errors.New("ошибка вывода данных")
	}
	return nil
}

type BankDataSecretViewer struct {
	SecretBankData
}

func (v *BankDataSecretViewer) ProvideDataToUser(
	smt domain.SecretsMetadata,
	data io.ReadCloser,
	promter sharedCli.Prompter) error {
	defer data.Close()
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return errors.New("ошибка чтения секрета")
	}
	err = json.Unmarshal(dataBytes, v)
	if err != nil {
		return errors.New("парсинга данных секрета")
	}
	err = promter.Send(fmt.Sprintf("номер карты: %s\n", v.CardNum))
	err = promter.Send(fmt.Sprintf("cvc код: %s\n", v.CVC))
	if err != nil {
		return errors.New("ошибка вывода данных")
	}
	return nil
}

type AuthDataSecretViewer struct {
	SecretAuthData
}

func (v *AuthDataSecretViewer) ProvideDataToUser(
	smt domain.SecretsMetadata,
	data io.ReadCloser,
	promter sharedCli.Prompter) error {
	defer data.Close()
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return errors.New("ошибка чтения секрета")
	}
	err = json.Unmarshal(dataBytes, v)
	if err != nil {
		return errors.New("парсинга данных секрета")
	}
	err = promter.Send(fmt.Sprintf("логин: %s\n", v.Login))
	err = promter.Send(fmt.Sprintf("пароль: %s\n", v.Password))
	if err != nil {
		return errors.New("ошибка вывода данных")
	}
	return nil
}

type BlobSecretViewer struct {
	DataDir string
}

func (v *BlobSecretViewer) ProvideDataToUser(
	smt domain.SecretsMetadata,
	data io.ReadCloser,
	promter sharedCli.Prompter) error {
	path := filepath.Join(v.DataDir, smt.SecretName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.New("ошибка открытия файла для записи")
	}
	defer file.Close()
	defer data.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return errors.New("ошибка записи данных в файл")
	}
	err = file.Close()
	if err != nil {
		return errors.New("ошибка закрытия файла после записи")
	}
	err = promter.Send(fmt.Sprintf("данные записаны в файл %s\n", path))
	if err != nil {
		return errors.New("ошибка вывода данных")
	}
	return nil
}
