package authcontextmanager

import (
	"context"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
)

type ctxKey int

const (
	userIDKey ctxKey = iota
)

func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
	userID, ok := ctx.Value(userIDKey).(domain.UserID)

	return userID, ok
}

func ContextWithUserID(ctx context.Context, userID domain.UserID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
