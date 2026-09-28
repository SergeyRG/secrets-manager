package mem

import (
	"context"
	"slices"
	"strings"

	secretsDomain "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/domain"
	secretsUseCases "github.com/SergeyRG/secrets-manager/client/internal/app/secrets/usecases"
)

type InMemoryMetadataRepository struct {
	storage           map[string]map[int]secretsDomain.SecretsMetadata
	latest            map[string]int
	sortedSecretsName []string
}

func NewInMemoryMetadataRepository(
	storage map[string]map[int]secretsDomain.SecretsMetadata,
	latest map[string]int,
	sortedSecretsName []string,
) *InMemoryMetadataRepository {
	return &InMemoryMetadataRepository{
		storage:           storage,
		latest:            latest,
		sortedSecretsName: sortedSecretsName,
	}
}

func (r *InMemoryMetadataRepository) AddSecretMetadata(
	ctx context.Context,
	sm secretsDomain.SecretsMetadata,
) error {
	if r.storage[sm.SecretName] == nil {
		r.storage[sm.SecretName] = make(map[int]secretsDomain.SecretsMetadata)
	}
	r.storage[sm.SecretName][int(sm.Version)] = sm

	_, exists := r.latest[sm.SecretName]

	if r.latest[sm.SecretName] < int(sm.Version) {
		r.latest[sm.SecretName] = int(sm.Version)
	}

	if !exists {
		nameToLower := strings.ToLower(sm.SecretName)

		index, _ := slices.BinarySearchFunc(r.sortedSecretsName, nameToLower, func(current string, target string) int {
			return strings.Compare(strings.ToLower(current), target)
		})

		r.sortedSecretsName = slices.Insert(r.sortedSecretsName, index, sm.SecretName)
	}

	return nil
}

func (r *InMemoryMetadataRepository) GetUserSecretMetadataByName(
	ctx context.Context,
	sName string,
	version int,
) (secretsDomain.SecretsMetadata, error) {
	sVersionsMap, ok := r.storage[sName]
	if !ok {
		return secretsDomain.SecretsMetadata{}, secretsUseCases.ErrNotFound
	}
	if version == 0 {
		version = r.latest[sName]
	}
	sm, ok := sVersionsMap[version]
	if !ok {
		return secretsDomain.SecretsMetadata{}, secretsUseCases.ErrIncorectSecretVersion
	}
	return sm, nil
}

func (r *InMemoryMetadataRepository) GetUserSecretsMetadataPage(
	ctx context.Context,
	page int,
	perPage int,
) ([]secretsDomain.SecretsMetadata, error) {

	totalSecrets := len(r.sortedSecretsName)
	offset := (page - 1) * perPage

	if offset >= totalSecrets || offset < 0 {
		return nil, nil
	}

	end := offset + perPage
	if end > totalSecrets {
		end = totalSecrets
	}

	pageSize := end - offset
	result := make([]secretsDomain.SecretsMetadata, 0, pageSize)

	for i := offset; i < end; i++ {
		name := r.sortedSecretsName[i]
		maxVersion := r.latest[name]

		metadata := r.storage[name][maxVersion]
		result = append(result, metadata)
	}

	return result, nil
}
