package credential

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

type Store struct {
	credRepo *repo.CredentialRepo
	key      []byte
}

func NewStore(credRepo *repo.CredentialRepo, key []byte) *Store {
	return &Store{credRepo: credRepo, key: key}
}

func (s *Store) Create(ctx context.Context, tenantID, name, code string, credType domain.CredentialType, config json.RawMessage, timeoutSecs int) (*domain.Credential, error) {
	preview := GeneratePreview(config)
	encrypted, err := Encrypt(config, s.key)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	c := &domain.Credential{
		ID:            generateID("cred_", 12),
		TenantID:      tenantID,
		Name:          name,
		Code:          code,
		Type:          credType,
		Config:        encrypted,
		ConfigPreview: preview,
		TimeoutSecs:   timeoutSecs,
		Status:        domain.CredStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.credRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	c.Config = nil
	return c, nil
}

func (s *Store) Get(ctx context.Context, id, tenantID string) (*domain.Credential, error) {
	c, err := s.credRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	c.Config = nil
	return c, nil
}

func (s *Store) GetDecrypted(ctx context.Context, id, tenantID string) (*domain.Credential, json.RawMessage, error) {
	c, err := s.credRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}
	plaintext, err := Decrypt(c.Config, s.key)
	if err != nil {
		return nil, nil, err
	}
	c.Config = nil
	return c, plaintext, nil
}

func (s *Store) GetDecryptedByCode(ctx context.Context, code, tenantID string) (*domain.Credential, json.RawMessage, error) {
	c, err := s.credRepo.GetByCode(ctx, code, tenantID)
	if err != nil {
		return nil, nil, err
	}
	plaintext, err := Decrypt(c.Config, s.key)
	if err != nil {
		return nil, nil, err
	}
	c.Config = nil
	return c, plaintext, nil
}

func (s *Store) List(ctx context.Context, tenantID, status string, limit, offset int) ([]*domain.Credential, int, error) {
	return s.credRepo.ListByTenant(ctx, tenantID, status, limit, offset)
}

func (s *Store) Update(ctx context.Context, id, tenantID string, name *string, config json.RawMessage, timeoutSecs *int) (*domain.Credential, error) {
	c, err := s.credRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if name != nil {
		c.Name = *name
	}
	if config != nil {
		c.ConfigPreview = GeneratePreview(config)
		encrypted, err := Encrypt(config, s.key)
		if err != nil {
			return nil, err
		}
		c.Config = encrypted
	}
	if timeoutSecs != nil {
		c.TimeoutSecs = *timeoutSecs
	}
	c.UpdatedAt = time.Now()

	if err := s.credRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	c.Config = nil
	return c, nil
}

func (s *Store) Delete(ctx context.Context, id, tenantID string) error {
	return s.credRepo.Delete(ctx, id, tenantID)
}
