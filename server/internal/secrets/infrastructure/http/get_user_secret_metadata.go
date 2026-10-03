package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type getUserSecretMetadataResp struct {
	SecretName string `json:"secret_name"`
	Version    int64  `json:"version"`
	SecretType string `json:"secret_type"`
}

type getUserSecretMetadataHandler struct {
	uc *secretUseCases.GetUserSecretMetadataUseCase
}

func NewGetUserSecretMetadataHandler(uc *secretUseCases.GetUserSecretMetadataUseCase) *getUserSecretMetadataHandler {
	return &getUserSecretMetadataHandler{uc: uc}
}

func (h *getUserSecretMetadataHandler) Handle(rw http.ResponseWriter, req *http.Request) {
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

	encoder := json.NewEncoder(rw)

	sm, err := h.uc.Execute(
		req.Context(),
		uID,
		secretName,
		version,
	)
	if err != nil {
		http.Error(rw, "ошибка получения данных из БД", http.StatusInternalServerError)
		return
	}

	resp := getUserSecretMetadataResp{
		SecretName: sm.SecretName,
		Version:    sm.Version,
		SecretType: sm.SecretType.ToString(),
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	encoder.Encode(resp)
}
