package http

import (
	"io"
	"net/http"
	"strings"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	"github.com/SergeyRG/secrets-manager/server/internal/security/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/security/use-cases"
)

type SetUserEncryptedKeyHandler struct {
	repo usecases.EncryptedKeyRepo
}

func NewSetUserEncryptedKeyHandler(repo usecases.EncryptedKeyRepo) *SetUserEncryptedKeyHandler {
	return &SetUserEncryptedKeyHandler{repo: repo}
}

func (h *SetUserEncryptedKeyHandler) Handle(rw http.ResponseWriter, req *http.Request) {
	uID, ok := authcontextmanager.UserIDFromContext(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.Contains(req.Header.Get("content-type"), "text/plain") {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	ek, err := io.ReadAll(io.LimitReader(req.Body, 1024*1024))
	if err != nil {
		http.Error(rw, "ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}

	err = h.repo.SetUserEncryptedKey(
		req.Context(),
		uID,
		domain.EncryptedKey(ek),
	)
	if err != nil {
		http.Error(rw, "ошибка записи ключа в БД", http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}
