package auth_test

import (
	"context"
	"errors"
	"sync"

	"github.com/dungpd/seta/auth-service/internal/auth"
)

type mockRefreshRepo struct {
	mu         sync.Mutex
	tokens     map[string]*auth.RefreshToken
	revokedAll []string
}

func newMockRefreshRepo() *mockRefreshRepo {
	return &mockRefreshRepo{tokens: make(map[string]*auth.RefreshToken)}
}

func (m *mockRefreshRepo) Insert(_ context.Context, t *auth.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t
	m.tokens[t.JTI] = &cp
	return nil
}

func (m *mockRefreshRepo) MarkRevoked(_ context.Context, jti string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[jti]
	if !ok {
		return errors.New("token not found: " + jti)
	}
	t.Revoked = true
	return nil
}

func (m *mockRefreshRepo) IsValid(_ context.Context, jti string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[jti]
	if !ok {
		return false, nil
	}
	return !t.Revoked, nil
}

func (m *mockRefreshRepo) RevokeAllForUser(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revokedAll = append(m.revokedAll, userID)
	for _, t := range m.tokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}
