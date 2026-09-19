package crypto

import (
	"errors"
	"fmt"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	"github.com/golang-jwt/jwt/v4"
)

type JWTManager struct {
	secretKey []byte
}

func NewJWTManager(secretKey []byte) *JWTManager {
	return &JWTManager{secretKey: secretKey}
}

var ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
var ErrTokenIsNotValid = errors.New("token is not valid")

type Claims struct {
	jwt.RegisteredClaims
	UserID domain.UserID
}

func (jwtm JWTManager) GenerateJWTAuthToken(userID domain.UserID) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
	})

	tokenString, err := token.SignedString(jwtm.secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (jwtm JWTManager) ValidateAndParseJWTAuthToken(tokenString string) (domain.UserID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, t.Header["alg"])
			}
			return jwtm.secretKey, nil
		})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", ErrTokenIsNotValid
	}

	return claims.UserID, nil
}
