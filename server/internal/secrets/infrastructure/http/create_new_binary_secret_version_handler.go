package http

import (
	"io"
	"net/http"
	"strings"

	"github.com/SergeyRG/secrets-manager/server/internal/app"
	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
)

type CreateNewBinarySecretVersionHandler struct {
	CreateNewBinarySecretVersionOrch *app.CreateNewBinarySecretVersionOrch
}

func NewCreateNewBinarySecretVersionHandler(o *app.CreateNewBinarySecretVersionOrch) *CreateNewBinarySecretVersionHandler {
	return &CreateNewBinarySecretVersionHandler{CreateNewBinarySecretVersionOrch: o}
}

func (h *CreateNewBinarySecretVersionHandler) Handle(rw http.ResponseWriter, req *http.Request) {
	uID, ok := authcontextmanager.UserIDFromContext(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !strings.Contains(req.Header.Get("content-type"), "multipart/form-data") {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	multipartReader, err := req.MultipartReader()
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var secretName string

	for {
		part, err := multipartReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		switch part.FormName() {
		case "secretName":
			val, err := io.ReadAll(part)
			part.Close()
			if err != nil {
				http.Error(rw, "неправильные имя секрета", http.StatusBadRequest)
				return
			}
			secretName = string(val)
		case "file":
			_, err := h.CreateNewBinarySecretVersionOrch.Execute(req.Context(), uID, secretName, part)
			part.Close()

			if err != nil {
				http.Error(rw, "ошибка создания бинарного секрета", http.StatusInternalServerError)
				return
			}

			rw.WriteHeader(http.StatusCreated)

			return

		default:
			part.Close()
		}
	}
}
