package mem_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	"github.com/SergeyRG/secrets-manager/client/internal/app/secrets/infra/mem"
	secretsUseCases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
)

// Хелпер для создания чистого репозитория перед каждым тест-кейсом
func setupRepo() *mem.InMemoryMetadataRepository {
	return mem.NewInMemoryMetadataRepository(
		make(map[string]map[int]secretsDomain.SecretsMetadata),
		make(map[string]int),
		make([]string, 0),
	)
}

func TestInMemoryMetadataRepository_AddAndGetSecretMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("Успешное добавление и получение актуальной версии (version=0)", func(t *testing.T) {
		repo := setupRepo()
		sm := secretsDomain.SecretsMetadata{SecretName: "Token", SecretType: secretsDomain.SecretTypeFreeText, Version: 1}

		if err := repo.AddSecretMetadata(ctx, sm); err != nil {
			t.Fatalf("не ожидали ошибки при добавлении: %v", err)
		}

		// Запрашиваем версию 0 (актуальную)
		res, err := repo.GetUserSecretMetadataByName(ctx, "Token", 0)
		if err != nil {
			t.Fatalf("ошибка при получении секретов: %v", err)
		}

		if res.Version != 1 || res.SecretName != "Token" {
			t.Errorf("вернулись неверные метаданные: %+v", res)
		}
	})

	t.Run("Успешное сохранение истории версий и обновление индекса latest", func(t *testing.T) {
		repo := setupRepo()
		sm1 := secretsDomain.SecretsMetadata{SecretName: "Pass", SecretType: secretsDomain.SecretTypeFreeText, Version: 1}
		sm2 := secretsDomain.SecretsMetadata{SecretName: "Pass", SecretType: secretsDomain.SecretTypeFreeText, Version: 2}

		_ = repo.AddSecretMetadata(ctx, sm1)
		_ = repo.AddSecretMetadata(ctx, sm2)

		// Проверяем, что версия 1 осталась доступна
		resV1, err := repo.GetUserSecretMetadataByName(ctx, "Pass", 1)
		if err != nil || resV1.Version != 1 {
			t.Errorf("ошибка чтения версии 1: %v", err)
		}

		// Проверяем, что версия 0 теперь указывает на максимальную (2)
		resLatest, err := repo.GetUserSecretMetadataByName(ctx, "Pass", 0)
		if err != nil || resLatest.Version != 2 {
			t.Errorf("ошибка чтения актуальной версии (latest): %v", err)
		}
	})

	t.Run("Ошибка: секрет с таким именем отсутствует", func(t *testing.T) {
		repo := setupRepo()
		_, err := repo.GetUserSecretMetadataByName(ctx, "Missing", 0)
		if !errors.Is(err, secretsUseCases.ErrNotFound) {
			t.Errorf("ожидалась ошибка ErrNotFound, получено: %v", err)
		}
	})

	t.Run("Ошибка: запрашиваемая версия отсутствует", func(t *testing.T) {
		repo := setupRepo()
		sm := secretsDomain.SecretsMetadata{SecretName: "Key", Version: 1}
		_ = repo.AddSecretMetadata(ctx, sm)

		_, err := repo.GetUserSecretMetadataByName(ctx, "Key", 99)
		if !errors.Is(err, secretsUseCases.ErrIncorectSecretVersion) {
			t.Errorf("ожидалась ошибка ErrIncorectSecretVersion, получено: %v", err)
		}
	})
}

func TestInMemoryMetadataRepository_GetUserSecretsMetadataPage(t *testing.T) {
	ctx := context.Background()

	// Инициализируем репозиторий набором данных для проверки сортировки и пагинации
	repo := setupRepo()

	// Добавляем имена в случайном порядке, чтобы проверить сортировку
	secrets := []secretsDomain.SecretsMetadata{
		{SecretName: "banana", Version: 1},
		{SecretName: "Apple", Version: 1},
		{SecretName: "cherry", Version: 1},
	}
	for _, s := range secrets {
		_ = repo.AddSecretMetadata(ctx, s)
	}

	tests := []struct {
		name     string
		page     int
		perPage  int
		wantList []secretsDomain.SecretsMetadata
	}{
		{
			name:    "Первая страница пагинации (алфавитный порядок, без учета регистра)",
			page:    1,
			perPage: 2,
			// Ожидаем сначала "Apple", затем "banana" (так как A -> B -> C)
			wantList: []secretsDomain.SecretsMetadata{
				{SecretName: "Apple", Version: 1},
				{SecretName: "banana", Version: 1},
			},
		},
		{
			name:    "Вторая страница пагинации (остаток)",
			page:    2,
			perPage: 2,
			wantList: []secretsDomain.SecretsMetadata{
				{SecretName: "cherry", Version: 1},
			},
		},
		{
			name:     "Выход за границы общего количества записей (offset >= total)",
			page:     3,
			perPage:  2,
			wantList: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := repo.GetUserSecretsMetadataPage(ctx, tt.page, tt.perPage)
			if err != nil {
				t.Fatalf("не ожидалось ошибки: %v", err)
			}

			if len(list) == 0 && len(tt.wantList) == 0 {
				return
			}

			if !reflect.DeepEqual(list, tt.wantList) {
				t.Errorf("получен неверный срез данных:\n%+v\nожидали:\n%+v", list, tt.wantList)
			}
		})
	}
}
