package service

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/dungpd/seta/auth-service/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	Issuer          = "auth-service"
	Audience        = "seta"
	AccessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type Claims struct {
	Role string `json:"role"`
	Type string `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

type AuthService struct {
	refreshRepo RefreshTokenRepo
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	redis       *redis.Client
}

func NewAuthService(
	refreshRepo RefreshTokenRepo,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	rdb *redis.Client,
) *AuthService {
	return &AuthService{
		refreshRepo: refreshRepo,
		privateKey:  privateKey,
		publicKey:   publicKey,
		redis:       rdb,
	}
}

func (s *AuthService) PublicKey() *rsa.PublicKey {
	return s.publicKey
}

// GenerateTokenPair issues an access + refresh token pair and persists the refresh token.
func (s *AuthService) GenerateTokenPair(userID, role string) (accessToken, refreshToken string, err error) {
	now := time.Now()

	accessClaims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
		},
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(s.privateKey)
	if err != nil {
		return
	}

	refreshJTI := uuid.NewString()
	refreshClaims := Claims{
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        refreshJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
		},
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims).SignedString(s.privateKey)
	if err != nil {
		return
	}

	err = s.refreshRepo.Insert(&model.RefreshToken{
		JTI:       refreshJTI,
		UserID:    userID,
		ExpiresAt: now.Add(refreshTokenTTL),
	})
	return
}

// RotateRefreshToken validates the refresh token, revokes it, and issues a new pair.
func (s *AuthService) RotateRefreshToken(tokenStr string) (accessToken, refreshToken string, err error) {
	claims, err := s.ParseToken(tokenStr,
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil || claims.Type != "refresh" {
		return "", "", fmt.Errorf("invalid refresh token")
	}

	valid, err := s.refreshRepo.IsValid(claims.ID)
	if err != nil {
		return "", "", err
	}
	if !valid {
		_ = s.refreshRepo.RevokeAllForUser(claims.Subject)
		return "", "", fmt.Errorf("refresh token reuse detected")
	}

	if err := s.refreshRepo.MarkRevoked(claims.ID); err != nil {
		return "", "", fmt.Errorf("failed to revoke token: %w", err)
	}

	return s.GenerateTokenPair(claims.Subject, claims.Role)
}

// RevokeSession blacklists the access token JTI and revokes the refresh token.
func (s *AuthService) RevokeSession(accessTokenStr, refreshTokenStr string) error {
	accessClaims, err := s.ParseToken(accessTokenStr,
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return fmt.Errorf("invalid access token: %w", err)
	}

	refreshClaims, err := s.ParseToken(refreshTokenStr)
	if err != nil {
		return fmt.Errorf("invalid refresh token: %w", err)
	}

	if err := s.refreshRepo.MarkRevoked(refreshClaims.ID); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	if ttl := time.Until(accessClaims.ExpiresAt.Time); ttl > 0 {
		return s.redis.Set(context.Background(), "jwt:blacklist:"+accessClaims.ID, "1", ttl).Err()
	}
	return nil
}

// ParseToken verifies the signature and returns parsed claims.
func (s *AuthService) ParseToken(tokenStr string, opts ...jwt.ParserOption) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, s.keyFunc(), opts...)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (s *AuthService) keyFunc() jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	}
}

func (s *AuthService) IsBlacklisted(jti string) (bool, error) {
	err := s.redis.Get(context.Background(), "jwt:blacklist:"+jti).Err()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
