package usecases_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
	mocks "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases/mocks"
	sharedUsecases "github.com/SergeyRG/secrets-manager/client/internal/app/shared/usecases"

	"go.uber.org/mock/gomock"
)

func TestGetSecretsListUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	page, perPage := 1, 10

	testMetas := []secretsDomain.SecretsMetadata{
		{SecretName: "secret_1", SecretType: secretsDomain.SecretTypeFreeText, Version: 1},
		{SecretName: "secret_2", SecretType: secretsDomain.SecretTypeBinary, Version: 2},
	}

	tests := []struct {
		name        string
		sessionType sharedUsecases.SessionType
		mockRepo    func(remote *mocks.MockSecretMetadataRepository, local *mocks.MockSecretMetadataRepository)
		wantList    []secretsDomain.SecretsMetadata
		wantErr     error
	}{
		{
			name:        "Режим Local: чтение только из локального кэша",
			sessionType: sharedUsecases.SessionTypeLocal,
			mockRepo: func(remote *mocks.MockSecretMetadataRepository, local *mocks.MockSecretMetadataRepository) {
				local.EXPECT().
					GetUserSecretsMetadataPage(ctx, page, perPage).
					Return(testMetas, nil).
					Times(1)
			},
			wantList: testMetas,
			wantErr:  nil,
		},
		{
			name:        "Режим Remote: успешный ответ сервера и сохранение данных в кэш",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(remote *mocks.MockSecretMetadataRepository, local *mocks.MockSecretMetadataRepository) {
				remote.EXPECT().
					GetUserSecretsMetadataPage(ctx, page, perPage).
					Return(testMetas, nil).
					Times(1)

				for _, item := range testMetas {
					local.EXPECT().
						AddSecretMetadata(ctx, item).
						Return(nil).
						Times(1)
				}
			},
			wantList: testMetas,
			wantErr:  nil,
		},
		{
			name:        "Режим Remote: сервер недоступен, данные берутся из кэша с маркером ошибки",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(remote *mocks.MockSecretMetadataRepository, local *mocks.MockSecretMetadataRepository) {
				remote.EXPECT().
					GetUserSecretsMetadataPage(ctx, page, perPage).
					Return(nil, usecases.ErrServerUnavailable).
					Times(1)

				local.EXPECT().
					GetUserSecretsMetadataPage(ctx, page, perPage).
					Return(testMetas, nil).
					Times(1)
			},
			wantList: testMetas,
			wantErr:  usecases.ErrDataFromLocalCache,
		},
		{
			name:        "Режим Remote: критическая ошибка сервера пробрасывается наружу без кэша",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(remote *mocks.MockSecretMetadataRepository, local *mocks.MockSecretMetadataRepository) {
				criticalErr := errors.New("unauthorized 401")
				remote.EXPECT().
					GetUserSecretsMetadataPage(ctx, page, perPage).
					Return(nil, criticalErr).
					Times(1)
			},
			wantList: nil,
			wantErr:  errors.New("unauthorized 401"),
		},
		{
			name:        "Ошибка чтения локального кэша при упавшем сервере",
			sessionType: sharedUsecases.SessionTypeRemote,
			mockRepo: func(remote *mocks.MockSecretMetadataRepository, local *mocks.MockSecretMetadataRepository) {
				remote.EXPECT().
					GetUserSecretsMetadataPage(ctx, page, perPage).
					Return(nil, usecases.ErrServerUnavailable).
					Times(1)

				local.EXPECT().
					GetUserSecretsMetadataPage(ctx, page, perPage).
					Return(nil, errors.New("db disk corruption")).
					Times(1)
			},
			wantList: nil,
			wantErr:  errors.New("db disk corruption"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRemote := mocks.NewMockSecretMetadataRepository(ctrl)
			mockLocal := mocks.NewMockSecretMetadataRepository(ctrl)

			tt.mockRepo(mockRemote, mockLocal)

			uc := usecases.NewGetSecretsListUseCase(mockRemote, mockLocal, tt.sessionType)

			list, err := uc.Execute(ctx, page, perPage)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("ожидалась ошибка %v, но получен nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("получена ошибка: %q, ожидалась: %q", err.Error(), tt.wantErr.Error())
				}
			} else if err != nil {
				t.Fatalf("не ожидалось ошибки, но получена: %v", err)
			}

			if !reflect.DeepEqual(list, tt.wantList) {
				t.Errorf("получен некорректный список:\n%+v\nожидалось:\n%+v", list, tt.wantList)
			}
		})
	}
}
