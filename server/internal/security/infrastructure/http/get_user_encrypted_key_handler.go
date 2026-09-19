package http

import (
	"errors"
	"net/http"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/security/use-cases"
)

type GetUserEncryptedKeyHandler struct {
	repo usecases.EncryptedKeyRepo
}

func NewGetUserEncryptedKeyHandler(repo usecases.EncryptedKeyRepo) *GetUserEncryptedKeyHandler {
	return &GetUserEncryptedKeyHandler{repo: repo}
}

func (h *GetUserEncryptedKeyHandler) Handle(rw http.ResponseWriter, req *http.Request) {
	uID, ok := authcontextmanager.UserIDFromContext(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	ek, err := h.repo.GetUserEncryptedKey(
		req.Context(),
		uID,
	)
	if err != nil {
		if errors.Is(usecases.ErrEncryptionKeyDoesntExists, err) {
			http.Error(rw, "ключ пользователя не задан", http.StatusNotFound)
			return
		}
		http.Error(rw, "ошибка получения ключа", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(ek))
}
