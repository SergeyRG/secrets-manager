package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SergeyRG/secrets-manager/client/internal/app/security/domain"
	domainMocks "github.com/SergeyRG/secrets-manager/client/internal/app/security/domain/mocks"
	"github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/security/usecases/mocks"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"go.uber.org/mock/gomock"
)

func TestCreateKeyUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	testPassword := "super_secret_passwd"
	testEncryptedKey := []byte("encrypted_master_key_bytes")

	tests := []struct {
		name         string
		sessionType  sharedUsecases.SessionType
		mockBehavior func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider)
		wantErr      error
	}{
		{
			name:        "Ошибка: создание ключа в локальной сессии запрещено",
			sessionType: sharedUsecases.SessionTypeLocal,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				// Метод прерывается до вызова зависимостей
			},
			wantErr: errors.New("создание ключа в локальной сессии недопустимо"),
		},
		{
			name:        "Ошибка: сбой провайдера паролей при вводе",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				pp.EXPECT().CreatePassword(ctx).Return("", errors.New("terminal close")).Times(1)
			},
			wantErr: errors.New("terminal close"),
		},
		{
			name:        "Успешный сценарий: генерация, шифрование и сохранение везде",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				pp.EXPECT().CreatePassword(ctx).Return(testPassword, nil).Times(1)
				remote.EXPECT().SaveEncryptedKey(ctx, testEncryptedKey).Return(nil).Times(1)
				local.EXPECT().SaveEncryptedKey(ctx, testEncryptedKey).Return(nil).Times(1)
			},
			wantErr: nil,
		},
		{
			name:        "Ошибка: удаленный репозиторий успешно сохранен, но сбой кэширования локально",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockBehavior: func(remote *domainMocks.MockEncryptedKeyRepository, local *domainMocks.MockEncryptedKeyRepository, pp *mocks.MockPasswdProvider) {
				pp.EXPECT().CreatePassword(ctx).Return(testPassword, nil).Times(1)
				remote.EXPECT().SaveEncryptedKey(ctx, testEncryptedKey).Return(nil).Times(1)
				local.EXPECT().SaveEncryptedKey(ctx, testEncryptedKey).Return(errors.New("disk full")).Times(1)
			},
			wantErr: usecases.ErrKeyDontSavedInCache,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRemote := domainMocks.NewMockEncryptedKeyRepository(ctrl)
			mockLocal := domainMocks.NewMockEncryptedKeyRepository(ctrl)
			mockPP := mocks.NewMockPasswdProvider(ctrl)

			tt.mockBehavior(mockRemote, mockLocal, mockPP)

			// Инициализируем KeyStorage с фейковой функцией шифрования ключа
			ks := domain.NewKeyStorage(
				nil, // KeyDecryptor
				func(data []byte, password string) ([]byte, error) {
					if password != testPassword {
						t.Errorf("в KeyEncryptor передан неверный пароль: %s", password)
					}
					return testEncryptedKey, nil
				},
				nil, // StreamEncryptor
				nil, // StreamDecryptor
			)

			uc := usecases.NewCreateKeyUseCase(mockRemote, mockLocal, tt.sessionType)

			err := uc.Execute(ctx, mockPP, ks)

			// Валидация ошибок
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но получен nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("получена ошибка: %q, ожидалась: %q", err.Error(), tt.wantErr.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}
		})
	}
}
