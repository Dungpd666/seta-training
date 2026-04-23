package service_test

import (
	"errors"
	"sync"

	"github.com/dungpd/seta/auth-service/internal/domain"
)

type mockUserRepo struct {
	mu        sync.Mutex
	users     []*domain.User
	createErr error
}

func (m *mockUserRepo) Create(u *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.users = append(m.users, &cp)
	return nil
}

func (m *mockUserRepo) FindByEmail(email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindAll() ([]domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.User, len(m.users))
	for i, u := range m.users {
		result[i] = *u
	}
	return result, nil
}

type mockRefreshTokenRepo struct {
	mu         sync.Mutex
	tokens     map[string]*domain.RefreshToken
	revokedAll []string
}

func newMockRefreshRepo() *mockRefreshTokenRepo {
	return &mockRefreshTokenRepo{tokens: make(map[string]*domain.RefreshToken)}
}

func (m *mockRefreshTokenRepo) Insert(t *domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t
	m.tokens[t.JTI] = &cp
	return nil
}

func (m *mockRefreshTokenRepo) MarkRevoked(jti string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[jti]
	if !ok {
		return errors.New("token not found: " + jti)
	}
	t.Revoked = true
	return nil
}

func (m *mockRefreshTokenRepo) IsValid(jti string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[jti]
	if !ok {
		return false, nil
	}
	return !t.Revoked, nil
}

func (m *mockRefreshTokenRepo) RevokeAllForUser(userID string) error {
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
