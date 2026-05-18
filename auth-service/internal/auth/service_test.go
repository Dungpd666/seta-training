package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/dungpd/seta/auth-service/internal/auth"
	"github.com/redis/go-redis/v9"
)

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

func newAuthSvc(t *testing.T) (auth.Service, *mockRefreshRepo) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := newMockRefreshRepo()
	return auth.NewService(repo, testPrivKey, testPubKey, rdb, "auth-service", "seta"), repo
}

func TestGenerateTokenPair_ClaimsCorrect(t *testing.T) {
	svc, repo := newAuthSvc(t)
	ctx := context.Background()

	accessToken, refreshToken, err := svc.GenerateTokenPair(ctx, "user-123", "member")
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
	if claims.Type != "access" {
		t.Errorf("access token type = %q, want %q", claims.Type, "access")
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
	ctx := context.Background()

	_, refreshToken, err := svc.GenerateTokenPair(ctx, "user-abc", "manager")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	newAccess, newRefresh, err := svc.RotateRefreshToken(ctx, refreshToken)
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
	valid, _ := repo.IsValid(ctx, oldClaims.ID)
	if valid {
		t.Error("old refresh token should be revoked after rotation")
	}
}

func TestRotateRefreshToken_KeepsRole(t *testing.T) {
	svc, _ := newAuthSvc(t)
	ctx := context.Background()

	_, refreshToken, err := svc.GenerateTokenPair(ctx, "user-1", "manager")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	newAccess, _, err := svc.RotateRefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("RotateRefreshToken: %v", err)
	}

	claims, err := svc.ParseToken(newAccess)
	if err != nil {
		t.Fatalf("ParseToken new access: %v", err)
	}
	if claims.Role != "manager" {
		t.Errorf("new access token role = %q, want %q", claims.Role, "manager")
	}
}

func TestRotateRefreshToken_ReuseDetected(t *testing.T) {
	svc, repo := newAuthSvc(t)
	ctx := context.Background()

	_, refreshToken, err := svc.GenerateTokenPair(ctx, "user-abc", "manager")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if _, _, err := svc.RotateRefreshToken(ctx, refreshToken); err != nil {
		t.Fatalf("first rotation: %v", err)
	}

	_, _, err = svc.RotateRefreshToken(ctx, refreshToken)
	if !errors.Is(err, auth.ErrTokenRevoked) {
		t.Errorf("expected ErrTokenRevoked, got: %v", err)
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
	ctx := context.Background()

	accessToken, _, err := svc.GenerateTokenPair(ctx, "user-xyz", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	_, _, err = svc.RotateRefreshToken(ctx, accessToken)
	if err == nil {
		t.Fatal("expected error when rotating with an access token, got nil")
	}
}

func TestRevokeSession_BlacklistsAccessToken(t *testing.T) {
	svc, _ := newAuthSvc(t)
	ctx := context.Background()

	accessToken, refreshToken, err := svc.GenerateTokenPair(ctx, "user-rev", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if err := svc.RevokeSession(ctx, accessToken, refreshToken); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	accessClaims, err := svc.ParseToken(accessToken)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	blacklisted, err := svc.IsBlacklisted(ctx, accessClaims.ID)
	if err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if !blacklisted {
		t.Error("access token JTI should be blacklisted after logout")
	}
}

func TestRevokeSession_RevokesRefreshToken(t *testing.T) {
	svc, repo := newAuthSvc(t)
	ctx := context.Background()

	accessToken, refreshToken, err := svc.GenerateTokenPair(ctx, "user-rev2", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	if err := svc.RevokeSession(ctx, accessToken, refreshToken); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}

	refreshClaims, err := svc.ParseToken(refreshToken)
	if err != nil {
		t.Fatalf("ParseToken refresh: %v", err)
	}

	valid, _ := repo.IsValid(ctx, refreshClaims.ID)
	if valid {
		t.Error("refresh token should be revoked after logout")
	}
}

func TestRevokeSession_RejectsCrossUserPair(t *testing.T) {
	svc, _ := newAuthSvc(t)
	ctx := context.Background()

	atA, _, err := svc.GenerateTokenPair(ctx, "user-A", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair A: %v", err)
	}
	_, rtB, err := svc.GenerateTokenPair(ctx, "user-B", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair B: %v", err)
	}

	err = svc.RevokeSession(ctx, atA, rtB)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("RevokeSession with mismatched subjects: err = %v, want ErrInvalidToken", err)
	}
}

func TestRevokeSession_RejectsAccessTokenAsRefresh(t *testing.T) {
	svc, _ := newAuthSvc(t)
	ctx := context.Background()

	at, _, err := svc.GenerateTokenPair(ctx, "user-1", "member")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}

	err = svc.RevokeSession(ctx, at, at)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("RevokeSession with access token as refresh: err = %v, want ErrInvalidToken", err)
	}
}

func TestIsBlacklisted_UnknownJTI(t *testing.T) {
	svc, _ := newAuthSvc(t)

	blacklisted, err := svc.IsBlacklisted(context.Background(), "nonexistent-jti")
	if err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if blacklisted {
		t.Error("unknown JTI should not be blacklisted")
	}
}
