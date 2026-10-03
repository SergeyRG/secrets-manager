package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretDomain "github.com/SergeyRG/secrets-manager/server/internal/secrets/domain"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type NewSecretTextDataReq struct {
	SecretName string `json:"secret_name"`
	SecretType string `json:"secret_type"`
	Data       string `json:"data"`
}

type CreateNewTextSecretHandler struct {
	CreateNewSecretUC *secretUseCases.CreateNewSecretUseCase
}

func NewCreateNewTextSecretHandler(uc *secretUseCases.CreateNewSecretUseCase) *CreateNewTextSecretHandler {
	return &CreateNewTextSecretHandler{CreateNewSecretUC: uc}
}

func (h *CreateNewTextSecretHandler) Handle(rw http.ResponseWriter, req *http.Request) {
	uID, ok := authcontextmanager.UserIDFromContext(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.Contains(req.Header.Get("content-type"), "application/json") {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	reqData := NewSecretTextDataReq{}

	reqBody, err := io.ReadAll(io.LimitReader(req.Body, 1024*1024))
	if err != nil {
		http.Error(rw, "ошибка чтения тела запроса", http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(reqBody, &reqData)
	if err != nil {
		http.Error(rw, "ошибка парсинга тела запроса", http.StatusBadRequest)
		return
	}

	_, err = h.CreateNewSecretUC.Execute(
		req.Context(),
		uID,
		reqData.SecretName,
		secretDomain.SecretTypeFromString(reqData.SecretType),
		io.NopCloser(strings.NewReader(string(reqData.Data))),
	)
	if err != nil {
		http.Error(rw, "ошибка создания секрета", http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusCreated)
}
