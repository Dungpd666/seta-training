package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dungpd/seta/auth-service/internal/user"
)

func TestImportFromCSV_AllSucceed(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	csv := `username,email,password,role
alice,alice@x.com,pass1,manager
bob,bob@x.com,pass2,member
charlie,charlie@x.com,pass3,member`

	result, err := svc.ImportFromCSV(context.Background(), strings.NewReader(csv), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Succeeded != 3 {
		t.Errorf("succeeded = %d, want 3", result.Succeeded)
	}
	if result.Failed != 0 {
		t.Errorf("failed = %d, want 0", result.Failed)
	}
	if len(result.Errors) != 0 {
		t.Errorf("errors = %v, want empty", result.Errors)
	}
}

func TestImportFromCSV_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	csv := `username,email,password,role
alice,alice@x.com,pass1,manager
bob,alice@x.com,pass2,member
charlie,charlie@x.com,pass3,member`

	result, err := svc.ImportFromCSV(context.Background(), strings.NewReader(csv), 1)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if result.Succeeded != 2 {
		t.Errorf("succeeded = %d, want 2", result.Succeeded)
	}
	if result.Failed != 1 {
		t.Errorf("failed = %d, want 1", result.Failed)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Row != 3 {
		t.Errorf("error row = %d, want 3 (bob's line)", result.Errors[0].Row)
	}
}

func TestImportFromCSV_ErrorsSorted(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	csv := `username,email,password,role
a,dup@x.com,p,member
b,dup@x.com,p,member
c,dup@x.com,p,member
d,dup@x.com,p,member
e,dup@x.com,p,member`

	result, _ := svc.ImportFromCSV(context.Background(), strings.NewReader(csv), 4)

	for i := 1; i < len(result.Errors); i++ {
		if result.Errors[i].Row < result.Errors[i-1].Row {
			t.Errorf("errors not sorted: %v", result.Errors)
			break
		}
	}
}

func TestImportFromCSV_InvalidHeader(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	csv := `name,mail,pwd,type
alice,alice@x.com,pass,member`

	_, err := svc.ImportFromCSV(context.Background(), strings.NewReader(csv), 2)
	if err == nil {
		t.Fatal("expected error for invalid header, got nil")
	}
}

func TestImportFromCSV_ContextCancel(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo)
	csv := `username,email,password,role
a,a@x.com,p,member
b,b@x.com,p,member
c,c@x.com,p,member`

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := svc.ImportFromCSV(ctx, strings.NewReader(csv), 1)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if result.Succeeded != 0 {
		t.Errorf("succeeded = %d, want 0 (ctx cancelled)", result.Succeeded)
	}
}

func TestImportFromCSV_DefaultWorkers(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo, user.WithWorkers(3))
	csv := `username,email,password,role
a,a@x.com,p,member`

	result, err := svc.ImportFromCSV(context.Background(), strings.NewReader(csv), 0)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if result.Succeeded != 1 {
		t.Errorf("succeeded = %d, want 1", result.Succeeded)
	}
}

func TestImportFromCSV_WithWorkersIgnoresNonPositive(t *testing.T) {
	repo := &mockRepo{}
	svc := user.NewService(repo, user.WithWorkers(-1), user.WithWorkers(0))
	csv := `username,email,password,role
a,a@x.com,p,member`

	result, err := svc.ImportFromCSV(context.Background(), strings.NewReader(csv), 0)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if result.Succeeded != 1 {
		t.Errorf("succeeded = %d, want 1 (default kept)", result.Succeeded)
	}
}
