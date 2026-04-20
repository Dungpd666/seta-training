package service_test

import (
	"testing"

	"github.com/dungpd/seta/auth-service/internal/model"
	"github.com/dungpd/seta/auth-service/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_HappyPath(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewUserService(repo)

	user, err := svc.Register("alice", "alice@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("username = %q, want %q", user.Username, "alice")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.Role != "member" {
		t.Errorf("role = %q, want %q", user.Role, "member")
	}
	if user.PasswordHash == "password123" {
		t.Error("password stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123")); err != nil {
		t.Errorf("password hash invalid: %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewUserService(repo)

	if _, err := svc.Register("alice", "alice@example.com", "pass123", "member"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, err := svc.Register("alice2", "alice@example.com", "pass456", "manager")
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}
	if err.Error() != "email already in use" {
		t.Errorf("error = %q, want %q", err.Error(), "email already in use")
	}
}

func TestRegister_RepoError(t *testing.T) {
	repo := &mockUserRepo{createErr: errBoom}
	svc := service.NewUserService(repo)

	_, err := svc.Register("bob", "bob@example.com", "pass123", "member")
	if err == nil {
		t.Fatal("expected repo error, got nil")
	}
}

func TestLogin_HappyPath(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewUserService(repo)

	if _, err := svc.Register("bob", "bob@example.com", "mypassword", "manager"); err != nil {
		t.Fatalf("register: %v", err)
	}

	user, err := svc.Login("bob@example.com", "mypassword")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if user.Email != "bob@example.com" {
		t.Errorf("email = %q, want %q", user.Email, "bob@example.com")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewUserService(repo)

	if _, err := svc.Register("carol", "carol@example.com", "correct", "member"); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := svc.Login("carol@example.com", "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("error = %q, want %q", err.Error(), "invalid credentials")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{}
	svc := service.NewUserService(repo)

	_, err := svc.Login("nobody@example.com", "pass")
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("error = %q, want %q", err.Error(), "invalid credentials")
	}
}

func TestListAll(t *testing.T) {
	repo := &mockUserRepo{
		users: []*model.User{
			{UserID: "1", Username: "a", Email: "a@x.com", Role: "member"},
			{UserID: "2", Username: "b", Email: "b@x.com", Role: "manager"},
		},
	}
	svc := service.NewUserService(repo)

	users, err := svc.ListAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("len = %d, want 2", len(users))
	}
}
