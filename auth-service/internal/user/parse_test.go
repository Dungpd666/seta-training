package user

import (
	"strings"
	"testing"
)

func TestNewCSVReader_HappyPath(t *testing.T) {
	input := strings.NewReader(`username,email,password,role
alice,alice@x.com,pass1,manager
bob,bob@x.com,pass2,member`)

	reader, err := newCSVReader(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	record, err := reader.Read()
	if err != nil {
		t.Fatalf("unexpected error reading row: %v", err)
	}
	if strings.TrimSpace(record[0]) != "alice" {
		t.Errorf("got %q, want alice", record[0])
	}
}

func TestNewCSVReader_CaseInsensitiveHeader(t *testing.T) {
	input := strings.NewReader(`USERNAME,Email,PASSWORD,Role
alice,alice@x.com,pass1,manager`)

	if _, err := newCSVReader(input); err != nil {
		t.Errorf("case-insensitive header should pass: %v", err)
	}
}

func TestNewCSVReader_WrongColumnCount(t *testing.T) {
	input := strings.NewReader(`username,email,password
alice,alice@x.com,pass1`)

	_, err := newCSVReader(input)
	if err == nil {
		t.Fatal("expected error for wrong column count, got nil")
	}
}

func TestNewCSVReader_WrongColumnName(t *testing.T) {
	input := strings.NewReader(`name,email,password,role
alice,alice@x.com,pass1,manager`)

	_, err := newCSVReader(input)
	if err == nil {
		t.Fatal("expected error for wrong column name, got nil")
	}
}

func TestNewCSVReader_EmptyFile(t *testing.T) {
	_, err := newCSVReader(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
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
