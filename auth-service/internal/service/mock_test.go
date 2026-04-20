package service_test

import (
	"errors"
	"sync"

	"github.com/dungpd/seta/auth-service/internal/model"
)

type mockUserRepo struct {
	mu        sync.Mutex
	users     []*model.User
	createErr error
}

func (m *mockUserRepo) Create(u *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.users = append(m.users, &cp)
	return nil
}

func (m *mockUserRepo) FindByEmail(email string) (*model.User, error) {
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

func (m *mockUserRepo) FindAll() ([]model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]model.User, len(m.users))
	for i, u := range m.users {
		result[i] = *u
	}
	return result, nil
}

type mockRefreshTokenRepo struct {
	mu         sync.Mutex
	tokens     map[string]*model.RefreshToken
	revokedAll []string
}

func newMockRefreshRepo() *mockRefreshTokenRepo {
	return &mockRefreshTokenRepo{tokens: make(map[string]*model.RefreshToken)}
}

func (m *mockRefreshTokenRepo) Insert(t *model.RefreshToken) error {
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
