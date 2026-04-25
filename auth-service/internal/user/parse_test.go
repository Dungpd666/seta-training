package user

import (
	"strings"
	"testing"
)

func TestParseCSV_HappyPath(t *testing.T) {
	input := strings.NewReader(`username,email,password,role
alice,alice@x.com,pass1,manager
bob,bob@x.com,pass2,member`)

	rows, err := parseCSV(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len = %d, want 2", len(rows))
	}
	if rows[0].LineNo != 2 {
		t.Errorf("rows[0].LineNo = %d, want 2", rows[0].LineNo)
	}
	if rows[0].Username != "alice" {
		t.Errorf("username = %q, want alice", rows[0].Username)
	}
	if rows[1].LineNo != 3 {
		t.Errorf("rows[1].LineNo = %d, want 3", rows[1].LineNo)
	}
}

func TestParseCSV_TrimWhitespace(t *testing.T) {
	input := strings.NewReader(`username,email,password,role
  alice  ,  alice@x.com  ,  pass1  ,  manager  `)

	rows, err := parseCSV(input)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rows[0].Username != "alice" {
		t.Errorf("username should be trimmed: %q", rows[0].Username)
	}
	if rows[0].Email != "alice@x.com" {
		t.Errorf("email should be trimmed: %q", rows[0].Email)
	}
	if rows[0].Role != "manager" {
		t.Errorf("role should be trimmed: %q", rows[0].Role)
	}
	if rows[0].Password != "pass1  " {
		t.Errorf("password should NOT be right-trimmed: %q", rows[0].Password)
	}
}

func TestParseCSV_CaseInsensitiveHeader(t *testing.T) {
	input := strings.NewReader(`USERNAME,Email,PASSWORD,Role
alice,alice@x.com,pass1,manager`)

	if _, err := parseCSV(input); err != nil {
		t.Errorf("case-insensitive header should pass: %v", err)
	}
}

func TestParseCSV_WrongColumnCount(t *testing.T) {
	input := strings.NewReader(`username,email,password
alice,alice@x.com,pass1`)

	_, err := parseCSV(input)
	if err == nil {
		t.Fatal("expected error for wrong column count, got nil")
	}
}

func TestParseCSV_WrongColumnName(t *testing.T) {
	input := strings.NewReader(`name,email,password,role
alice,alice@x.com,pass1,manager`)

	_, err := parseCSV(input)
	if err == nil {
		t.Fatal("expected error for wrong column name, got nil")
	}
}

func TestParseCSV_EmptyFile(t *testing.T) {
	_, err := parseCSV(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

func TestParseCSV_RowMissingColumn(t *testing.T) {
	input := strings.NewReader(`username,email,password,role
alice,alice@x.com,pass1`)

	_, err := parseCSV(input)
	if err == nil {
		t.Fatal("expected error for malformed row, got nil")
	}
}

func TestValidateHeader(t *testing.T) {
	tests := []struct {
		name    string
		header  []string
		wantErr bool
	}{
		{"correct", []string{"username", "email", "password", "role"}, false},
		{"uppercase", []string{"USERNAME", "EMAIL", "PASSWORD", "ROLE"}, false},
		{"with spaces", []string{" username ", " email ", " password ", " role "}, false},
		{"too few", []string{"username", "email", "password"}, true},
		{"too many", []string{"username", "email", "password", "role", "extra"}, true},
		{"wrong name", []string{"username", "email", "pwd", "role"}, true},
		{"wrong order", []string{"email", "username", "password", "role"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeader(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
