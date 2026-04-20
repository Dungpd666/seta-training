package service_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/dungpd/seta/auth-service/internal/service"
	"github.com/redis/go-redis/v9"
)

var errBoom = errors.New("boom")

var (
	testPrivKey *rsa.PrivateKey
	testPubKey  *rsa.PublicKey
)

func TestMain(m *testing.M) {
	var err error
	testPrivKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("rsa.GenerateKey: " + err.Error())
	}
	testPubKey = &testPrivKey.PublicKey
	os.Exit(m.Run())
}

func newAuthSvc(t *testing.T) (*service.AuthService, *mockRefreshTokenRepo) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := newMockRefreshRepo()
	return service.NewAuthService(repo, testPrivKey, testPubKey, rdb), repo
}

func TestGenerateTokenPair_ClaimsCorrect(t *testing.T) {
	svc, repo := newAuthSvc(t)

	accessToken, refreshToken, err := svc.GenerateTokenPair("user-123", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("empty token string")
	}

	claims, err := svc.ParseToken(accessToken)
	if err != nil {
		t.Fatalf("ParseToken access: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("sub = %q, want %q", claims.Subject, "user-123")
	}
	if claims.Role != "member" {
		t.Errorf("role = %q, want %q", claims.Role, "member")
	}
	if claims.Type != "" {
		t.Errorf("access token type = %q, want empty", claims.Type)
	}

	refreshClaims, err := svc.ParseToken(refreshToken)
	if err != nil {
		t.Fatalf("ParseToken refresh: %v", err)
	}
	if refreshClaims.Type != "refresh" {
		t.Errorf("refresh token type = %q, want %q", refreshClaims.Type, "refresh")
	}

	repo.mu.Lock()
	count := len(repo.tokens)
	repo.mu.Unlock()
	if count != 1 {
		t.Errorf("refresh tokens in repo = %d, want 1", count)
	}
}

func TestRotateRefreshToken_HappyPath(t *testing.T) {
	svc, repo := newAuthSvc(t)

	_, refreshToken, err := svc.GenerateTokenPair("user-abc", "manager")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	newAccess, newRefresh, err := svc.RotateRefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("RotateRefreshToken: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Error("empty tokens after rotation")
	}
	if newRefresh == refreshToken {
		t.Error("refresh token was not rotated — got same token back")
	}

	oldClaims, err := svc.ParseToken(refreshToken)
	if err != nil {
		t.Fatalf("ParseToken old refresh: %v", err)
	}
	valid, _ := repo.IsValid(oldClaims.ID)
	if valid {
		t.Error("old refresh token should be revoked after rotation")
	}
}

func TestRotateRefreshToken_ReuseDetected(t *testing.T) {
	svc, repo := newAuthSvc(t)

	_, refreshToken, err := svc.GenerateTokenPair("user-abc", "manager")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if _, _, err := svc.RotateRefreshToken(refreshToken); err != nil {
		t.Fatalf("first rotation: %v", err)
	}

	_, _, err = svc.RotateRefreshToken(refreshToken)
	if err == nil {
		t.Fatal("expected reuse error, got nil")
	}
	if err.Error() != "refresh token reuse detected" {
		t.Errorf("error = %q, want %q", err.Error(), "refresh token reuse detected")
	}

	repo.mu.Lock()
	called := len(repo.revokedAll) > 0
	repo.mu.Unlock()
	if !called {
		t.Error("RevokeAllForUser should be called on reuse detection")
	}
}

func TestRotateRefreshToken_AccessTokenRejected(t *testing.T) {
	svc, _ := newAuthSvc(t)

	accessToken, _, err := svc.GenerateTokenPair("user-xyz", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	_, _, err = svc.RotateRefreshToken(accessToken)
	if err == nil {
		t.Fatal("expected error when rotating with an access token, got nil")
	}
}

func TestRevokeSession_BlacklistsAccessToken(t *testing.T) {
	svc, _ := newAuthSvc(t)

	accessToken, refreshToken, err := svc.GenerateTokenPair("user-rev", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if err := svc.RevokeSession(accessToken, refreshToken); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	accessClaims, err := svc.ParseToken(accessToken)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	blacklisted, err := svc.IsBlacklisted(accessClaims.ID)
	if err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if !blacklisted {
		t.Error("access token JTI should be blacklisted after logout")
	}
}

func TestRevokeSession_RevokesRefreshToken(t *testing.T) {
	svc, repo := newAuthSvc(t)

	accessToken, refreshToken, err := svc.GenerateTokenPair("user-rev2", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if err := svc.RevokeSession(accessToken, refreshToken); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	refreshClaims, err := svc.ParseToken(refreshToken)
	if err != nil {
		t.Fatalf("ParseToken refresh: %v", err)
	}

	valid, _ := repo.IsValid(refreshClaims.ID)
	if valid {
		t.Error("refresh token should be revoked after logout")
	}
}

func TestIsBlacklisted_UnknownJTI(t *testing.T) {
	svc, _ := newAuthSvc(t)

	blacklisted, err := svc.IsBlacklisted("nonexistent-jti")
	if err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if blacklisted {
		t.Error("unknown JTI should not be blacklisted")
	}
}
