package commandhandlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	sharedCli "github.com/SergeyRG/secrets-manager/client/internal/app/shared/infra/cli"

	"resty.dev/v3"
)

type SecretListEntry struct {
	SecretName string `json:"secret_name"`
	Version    int    `json:"version"`
	Error      string `json:"error"`
}

type GetUserSecretsListHandler struct {
	restyClient *resty.Client
	relativeURL string
	perPage     int
}

func NewGetUserSecretsListHandler(restyClient *resty.Client, relativeURL string, perPage int) *GetUserSecretsListHandler {
	return &GetUserSecretsListHandler{restyClient: restyClient, relativeURL: relativeURL, perPage: perPage}
}

func (h *GetUserSecretsListHandler) Handle(ctx context.Context, args []string, prompter sharedCli.Prompter) error {
	page := 1

OuterLoop:
	for {
		response, err := h.restyClient.
			R().
			SetContext(ctx).
			SetResponseDoNotParse(true).
			SetQueryParams(map[string]string{
				"page":     strconv.Itoa(page),
				"per_page": strconv.Itoa(h.perPage),
			}).Get(h.relativeURL)

		if err != nil {

			return errors.New("ошибка направляния запроса на сервер")
		}

		bodyStream := response.Body

		if response.IsStatusFailure() {
			bodyStream.Close()
			return fmt.Errorf("server returned error status: %s", response.Status())
		}

		scanner := bufio.NewScanner(bodyStream)
		hasData := false
		count := 0
		for scanner.Scan() {
			hasData = true
			lineBytes := scanner.Bytes()

			if len(bytes.TrimSpace(lineBytes)) == 0 {
				continue
			}

			var secretItem SecretListEntry
			if err := json.Unmarshal(lineBytes, &secretItem); err != nil {
				prompter.Send(fmt.Sprintf("ошибка парсинга строки: %v\n", err))
				continue
			}

			prompter.Send(fmt.Sprintf("\tКлюч: %s\t версия: %v\t\n", secretItem.SecretName, secretItem.Version))
			count++
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("ошибка во время чтения сетевого потока: %w", err)
		}

		if count < h.perPage {
			return nil
		}

		if !hasData {
			return nil
		}

		prompter.Send("\n")
		for {
			prompter.Send("Для загрузки следующей страницы нажмите 'n', для завершения нажмите 'e': ")
			m, err := prompter.Receive(false)
			if err != nil {
				return fmt.Errorf("ошибка во время чтения пользовательского ввода: %w", err)
			}
			if m == "n" {
				page++
				continue OuterLoop
			} else if m == "e" {
				return nil
			}
		}
	}
}
