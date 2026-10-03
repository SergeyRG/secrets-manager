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

func TestRegisterUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		login         string
		password      string
		mockSetup     func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher)
		expectError   bool
		expectedError error
	}{
		{
			name:     "Успешная регистрация",
			login:    "valid_login_123",
			password: "secure_password",
			mockSetup: func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher) {
				hasher.EXPECT().HashPassword("secure_password").Return("hashed_pw", nil)
				repo.EXPECT().AddUser(ctx, gomock.Any()).Return(nil)
			},
			expectError:   false,
			expectedError: nil,
		},
		{
			name:     "Ошибка хэширования пароля",
			login:    "valid_login",
			password: "secure_password",
			mockSetup: func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher) {
				hasher.EXPECT().HashPassword("secure_password").Return("", errors.New("hashing failed"))
			},
			expectError:   true,
			expectedError: errors.New("hashing failed"),
		},
		{
			name:     "Ошибка валидации логина (недопустимые символы)",
			login:    "invalid-login-!!!",
			password: "secure_password",
			mockSetup: func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher) {
				hasher.EXPECT().HashPassword("secure_password").Return("hashed_pw", nil)
			},
			expectError:   true,
			expectedError: authDomain.ErrInvalidLoginFormat,
		},
		{
			name:     "Ошибка репозитория (логин занят)",
			login:    "taken_login",
			password: "secure_password",
			mockSetup: func(repo *mocks.MockUserRepo, hasher *mocks.MockHasher) {
				hasher.EXPECT().HashPassword("secure_password").Return("hashed_pw", nil)
				repo.EXPECT().AddUser(ctx, gomock.Any()).Return(usecases.ErrLoginBusy)
			},
			expectError:   true,
			expectedError: usecases.ErrLoginBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockUserRepo(ctrl)
			mockHasher := mocks.NewMockHasher(ctrl)

			tt.mockSetup(mockRepo, mockHasher)

			uc := usecases.NewRegisterUserUseCase(mockRepo, mockHasher)
			userID, err := uc.Execute(ctx, tt.login, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Empty(t, userID)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, userID)
				assert.IsType(t, domain.UserID(""), userID)
			}
		})
	}
}
