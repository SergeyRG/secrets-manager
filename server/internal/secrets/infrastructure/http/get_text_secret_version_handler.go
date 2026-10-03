package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type getTextSecretVersionResp struct {
	SecretName string `json:"secret_name"`
	Version    int    `json:"version"`
	Data       string `json:"data"`
}

type GetTextSecretVersionHandler struct {
	GetSecretVersionUC *secretUseCases.GetSecretVersionUseCase
}

func NewGetTextSecretVersionHandler(uc *secretUseCases.GetSecretVersionUseCase) *GetTextSecretVersionHandler {
	return &GetTextSecretVersionHandler{GetSecretVersionUC: uc}
}

func (h *GetTextSecretVersionHandler) Handle(rw http.ResponseWriter, req *http.Request) {
	uID, ok := authcontextmanager.UserIDFromContext(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	secretName := req.Header.Get("X-Secret-Name")
	versionStr := req.Header.Get("X-Secret-Version")

	if secretName == "" || versionStr == "" {
		http.Error(rw, "не заданы необходимые заголовки", http.StatusBadRequest)
		return
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		http.Error(rw, "версия должна быть целым числом", http.StatusBadRequest)
		return
	}

	dataStream, err := h.GetSecretVersionUC.Execute(
		req.Context(),
		uID,
		secretName,
		version,
	)
	if err != nil {
		http.Error(rw, "ошибка получения секрета", http.StatusInternalServerError)
		return
	}
	defer dataStream.Close()

	data, err := io.ReadAll(dataStream)
	if err != nil {
		http.Error(rw, "ошибка получения секрета", http.StatusInternalServerError)
		return
	}

	resp := getTextSecretVersionResp{
		SecretName: secretName,
		Version:    version,
		Data:       string(data),
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(resp)
}
