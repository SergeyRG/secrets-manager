package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/SergeyRG/secrets-manager/internal/shared/domain"
	authDomain "github.com/SergeyRG/secrets-manager/server/internal/auth/domain"
	usecases "github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases"
	"github.com/SergeyRG/secrets-manager/server/internal/auth/use-cases/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	testUser := &authDomain.User{
		UserID:  domain.UserID("test-user-id"),
		Login:   "valid_login",
		PwdHash: "hashed_password",
	}

	tests := []struct {
		name           string
		login          string
		password       string
		mockSetup      func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher)
		expectedUserID domain.UserID
		expectedError  error
	}{
		{
			name:     "Успешная авторизация",
			login:    "valid_login",
			password: "correct_password",
			mockSetup: func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher) {
				repo.EXPECT().GetUserByLogin(ctx, "valid_login").Return(testUser, nil)
				hasher.EXPECT().CheckPasswordHash("correct_password", "hashed_password").Return(true)
			},
			expectedUserID: testUser.UserID,
			expectedError:  nil,
		},
		{
			name:     "Пользователь не найден",
			login:    "unknown_login",
			password: "any_password",
			mockSetup: func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher) {
				repo.EXPECT().GetUserByLogin(ctx, "unknown_login").Return(nil, errors.New("user not found"))
			},
			expectedUserID: "",
			expectedError:  errors.New("user not found"),
		},
		{
			name:     "Неверный пароль",
			login:    "valid_login",
			password: "wrong_password",
			mockSetup: func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher) {
				repo.EXPECT().GetUserByLogin(ctx, "valid_login").Return(testUser, nil)
				hasher.EXPECT().CheckPasswordHash("wrong_password", "hashed_password").Return(false)
			},
			expectedUserID: "",
			expectedError:  usecases.ErrWrongLoginOrPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockUserRepo(ctrl)
			mockHasher := mocks.NewMockHasher(ctrl)

			tt.mockSetup(mockRepo, mockHasher)

			uc := usecases.NewAuthUserUseCase(mockRepo, mockHasher)
			userID, err := uc.Execute(ctx, tt.login, tt.password)

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedUserID, userID)
		})
	}
}
