package user_test

import (
	"context"
	"errors"
	"sync"

	"github.com/dungpd/seta/auth-service/internal/user"
)

var errBoom = errors.New("boom")

type mockRepo struct {
	mu        sync.Mutex
	users     []*user.User
	createErr error
}

func (m *mockRepo) Create(_ context.Context, u *user.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.users = append(m.users, &cp)
	return nil
}

func (m *mockRepo) FindByEmail(_ context.Context, email string) (*user.User, error) {
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

func (m *mockRepo) FindAll(_ context.Context) ([]user.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]user.User, len(m.users))
	for i, u := range m.users {
		result[i] = *u
	}
	return result, nil
}
