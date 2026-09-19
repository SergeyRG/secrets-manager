package middleware

import (
	"net/http"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	authcontextmanager "github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/auth_context_manager"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/infrastructure/crypto"
)

func NewAuthMiddleware(jwtm *crypto.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			cookieToken, err := r.Cookie("auth_token")
			var tokenString string
			var userID domain.UserID

			if err != nil {
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}
			tokenString = cookieToken.Value
			userID, err = jwtm.ValidateAndParseJWTAuthToken(tokenString)

			if err != nil || userID == "" {
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}

			ctx := authcontextmanager.ContextWithUserID(r.Context(), userID)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
