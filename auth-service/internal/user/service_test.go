package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dungpd/seta/auth-service/internal/user"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)

	u, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("username = %q, want %q", u.Username, "alice")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("email = %q, want %q", u.Email, "alice@example.com")
	}
	if u.Role != "member" {
		t.Errorf("role = %q, want %q", u.Role, "member")
	}
	if u.PasswordHash == "password123" {
		t.Error("password stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("password123")); err != nil {
		t.Errorf("password hash invalid: %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "alice", "alice@example.com", "pass123", "member"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, err := svc.Register(ctx, "alice2", "alice@example.com", "pass456", "manager")
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}
	if !errors.Is(err, user.ErrEmailInUse) {
		t.Errorf("error = %v, want ErrEmailInUse", err)
	}
}

func TestRegister_RepoError(t *testing.T) {
	repo := &mockRepo{createErr: errBoom}
	svc := user.NewService(repo)

	_, err := svc.Register(context.Background(), "bob", "bob@example.com", "pass123", "member")
	if err == nil {
		t.Fatal("expected repo error, got nil")
	}
}

func TestLogin_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "bob", "bob@example.com", "mypassword", "manager"); err != nil {
		t.Fatalf("register: %v", err)
	}

	u, err := svc.Login(ctx, "bob@example.com", "mypassword")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if u.Email != "bob@example.com" {
		t.Errorf("email = %q, want %q", u.Email, "bob@example.com")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "carol", "carol@example.com", "correct", "member"); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := svc.Login(ctx, "carol@example.com", "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	if !errors.Is(err, user.ErrInvalidCredentials) {
		t.Errorf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)

	_, err := svc.Login(context.Background(), "nobody@example.com", "pass")
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
	if !errors.Is(err, user.ErrInvalidCredentials) {
		t.Errorf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestListPage(t *testing.T) {
	repo := &mockRepo{
		users: []*user.User{
			{UserID: "1", Username: "a", Email: "a@x.com", Role: "member"},
			{UserID: "2", Username: "b", Email: "b@x.com", Role: "manager"},
		},
	}
	svc := user.NewService(repo)

	users, total, err := svc.ListPage(context.Background(), "", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(users) != 2 {
		t.Errorf("len = %d, want 2", len(users))
	}
}
