package http

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type GetBlobSecretVersionHandler struct {
	GetSecretVersionUC *secretUseCases.GetSecretVersionUseCase
}

func NewGetBlobSecretVersionHandler(uc *secretUseCases.GetSecretVersionUseCase) *GetBlobSecretVersionHandler {
	return &GetBlobSecretVersionHandler{GetSecretVersionUC: uc}
}

func (h *GetBlobSecretVersionHandler) Handle(rw http.ResponseWriter, req *http.Request) {
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

	rw.Header().Set("Content-Type", "application/octet-stream")
	rw.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.enc\"", secretName))
	rw.Header().Set("Accept-Ranges", "none")

	rw.WriteHeader(http.StatusOK)

	_, err = io.Copy(rw, dataStream)
	if err != nil {
		return
	}
}
