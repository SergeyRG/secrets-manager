package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	secretUseCases "github.com/SergeyRG/secrets-manager/server/internal/secrets/use-cases"
)

type getUserSecretMetadataPageResp struct {
	SecretName string `json:"secret_name"`
	SecretType string `json:"secret_type"`
	Version    int64  `json:"version"`
	Error      string `json:"error"`
}

type GetUserSecretMetadataPageHandler struct {
	GetUserSecretMetadataPageUC *secretUseCases.GetUserSecretMetadataPageUseCase
}

func NewGetUserSecretMetadataPageHandler(uc *secretUseCases.GetUserSecretMetadataPageUseCase) *GetUserSecretMetadataPageHandler {
	return &GetUserSecretMetadataPageHandler{GetUserSecretMetadataPageUC: uc}
}

func (h *GetUserSecretMetadataPageHandler) Handle(rw http.ResponseWriter, req *http.Request) {
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "стриминг не поддерживается", http.StatusInternalServerError)
		return
	}
	uID, ok := authcontextmanager.UserIDFromContext(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	query := req.URL.Query()

	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(query.Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = 20
	}

	encoder := json.NewEncoder(rw)
	headerWritten := false

	for sm, err := range h.GetUserSecretMetadataPageUC.Execute(
		req.Context(),
		uID,
		page,
		perPage,
	) {
		if err != nil && !headerWritten {
			http.Error(rw, "ошибка получения данных из БД", http.StatusInternalServerError)
			return
		}

		if err != nil && headerWritten {
			encoder.Encode(getUserSecretMetadataPageResp{
				SecretName: "",
				Version:    0,
				Error:      "ошибка получения информации об очередном секрете",
			})
			flusher.Flush()
			break
		}
		resp := getUserSecretMetadataPageResp{
			SecretName: sm.SecretName,
			SecretType: sm.SecretType.ToString(),
			Version:    sm.Version,
			Error:      "",
		}
		if !headerWritten {
			rw.Header().Set("Content-Type", "application/x-ndjson")
			rw.Header().Set("X-Content-Type-Options", "nosniff")
			rw.WriteHeader(http.StatusOK)
			headerWritten = true
		}
		err := encoder.Encode(resp)
		if err != nil {
			break
		}
		flusher.Flush()
	}
}
