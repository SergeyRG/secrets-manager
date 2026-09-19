package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/crypto"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases"
)

type RegistreHandler struct {
	RegistreUseCase *usecases.RegisterUserUseCase
	jwtm            *crypto.JWTManager
}

func NewRegistreHandler(registreUseCase *usecases.RegisterUserUseCase, jwtm *crypto.JWTManager) *RegistreHandler {
	return &RegistreHandler{RegistreUseCase: registreUseCase, jwtm: jwtm}
}

func (h RegistreHandler) Handle(rw http.ResponseWriter, req *http.Request) {
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	user := &usecases.UserAuthDTO{}
	req.Body = http.MaxBytesReader(rw, req.Body, 1024*1024)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, user)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if user.Login == "" || user.Password == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	uID, err := h.RegistreUseCase.Execute(req.Context(), user.Login, user.Password)

	if err != nil {
		if errors.Is(err, usecases.ErrLoginBusy) {
			rw.WriteHeader(http.StatusConflict)
			return
		}

		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokenString, err := h.jwtm.GenerateJWTAuthToken(uID)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(rw, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
	})

	rw.WriteHeader(http.StatusOK)
}
