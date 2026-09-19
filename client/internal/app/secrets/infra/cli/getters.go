package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"

	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"
)

type FreeTextSecretGetter struct {
	SecretFreeText
}

func (v *FreeTextSecretGetter) GetDataFromUser(
	promter sharedCli.Prompter) (io.ReadCloser, error) {
	err := promter.Send("\nВведите текстовые данные: ")
	if err != nil {
		return nil, errors.New("ошибка вывода данных")
	}
	UserData, err := promter.Receive(false)
	if err != nil {
		return nil, errors.New("ошибка ввода данных произвольного секрета")
	}
	v.Data = UserData
	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.New("ошибка кодирования данных произвольного секрета")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

type BankDataSecretGetter struct {
	SecretBankData
}

func (v *BankDataSecretGetter) GetDataFromUser(
	promter sharedCli.Prompter) (io.ReadCloser, error) {
	err := promter.Send("\nНомер банковской карты: ")
	if err != nil {
		return nil, errors.New("ошибка вывода данных")
	}
	cardNum, err := promter.Receive(false)
	if err != nil {
		return nil, errors.New("ошибка ввода номера банковской карты")
	}
	err = promter.Send("\nCVC код: ")
	if err != nil {
		return nil, errors.New("ошибка вывода данных")
	}
	cvc, err := promter.Receive(false)
	if err != nil {
		return nil, errors.New("ошибка ввода cvc")
	}
	v.CardNum = cardNum
	v.CVC = cvc
	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.New("ошибка кодирования данных секрета")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

type AuthDataSecretGetter struct {
	SecretAuthData
}

func (v *AuthDataSecretGetter) GetDataFromUser(
	promter sharedCli.Prompter) (io.ReadCloser, error) {
	err := promter.Send("\nлогин: ")
	if err != nil {
		return nil, errors.New("ошибка вывода данных")
	}
	login, err := promter.Receive(false)
	if err != nil {
		return nil, errors.New("ошибка ввода логина")
	}
	err = promter.Send("\nПароль: ")
	if err != nil {
		return nil, errors.New("ошибка вывода данных")
	}
	pswd, err := promter.Receive(false)
	if err != nil {
		return nil, errors.New("ошибка ввода пароля")
	}
	v.Login = login
	v.Password = pswd
	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.New("ошибка кодирования данных секрета")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

type BlobSecretGetter struct {
}

func (v *BlobSecretGetter) GetDataFromUser(
	promter sharedCli.Prompter) (io.ReadCloser, error) {
	err := promter.Send("\nПуть к файлу: ")
	if err != nil {
		return nil, errors.New("ошибка вывода данных")
	}
	path, err := promter.Receive(false)
	if err != nil {
		return nil, errors.New("ошибка ввода данных")
	}

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, errors.New("ошибка открытия файла")
	}
	return file, nil
}
