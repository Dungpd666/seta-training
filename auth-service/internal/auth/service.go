package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	AccessTokenTTL   = 15 * time.Minute
	refreshTokenTTL  = 7 * 24 * time.Hour
	accessTokenType  = "access"
	refreshTokenType = "refresh"
	KeyID            = "auth-service-key"
)

type Claims struct {
	Role string `json:"role"`
	Type string `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

type RefreshTokenRepo interface {
	Insert(ctx context.Context, rt *RefreshToken) error
	MarkRevoked(ctx context.Context, jti string) error
	IsValid(ctx context.Context, jti string) (bool, error)
	RevokeAllForUser(ctx context.Context, userID string) error
}

type Service interface {
	GenerateTokenPair(ctx context.Context, userID, role string) (string, string, error)
	RotateRefreshToken(ctx context.Context, tokenStr string) (string, string, error)
	RevokeSession(ctx context.Context, accessTokenStr, refreshTokenStr string) error
	ParseToken(tokenStr string, opts ...jwt.ParserOption) (*Claims, error)
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	PublicKey() *rsa.PublicKey
}

type service struct {
	refreshRepo RefreshTokenRepo
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	redis       *redis.Client
	issuer      string
	audience    string
}

func NewService(
	refreshRepo RefreshTokenRepo,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	rdb *redis.Client,
	issuer, audience string,
) Service {
	return &service{
		refreshRepo: refreshRepo,
		privateKey:  privateKey,
		publicKey:   publicKey,
		redis:       rdb,
		issuer:      issuer,
		audience:    audience,
	}
}

func (s *service) PublicKey() *rsa.PublicKey {
	return s.publicKey
}

func (s *service) GenerateTokenPair(ctx context.Context, userID, role string) (accessToken, refreshToken string, err error) {
	now := time.Now()

	accessClaims := Claims{
		Role: role,
		Type: accessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenObj.Header["kid"] = KeyID
	accessToken, err = accessTokenObj.SignedString(s.privateKey)
	if err != nil {
		return
	}

	refreshJTI := uuid.NewString()
	refreshClaims := Claims{
		Role: role,
		Type: refreshTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        refreshJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
		},
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenObj.Header["kid"] = KeyID
	refreshToken, err = refreshTokenObj.SignedString(s.privateKey)
	if err != nil {
		return
	}

	err = s.refreshRepo.Insert(ctx, &RefreshToken{
		JTI:       refreshJTI,
		UserID:    userID,
		ExpiresAt: now.Add(refreshTokenTTL),
	})
	return
}

func (s *service) RotateRefreshToken(ctx context.Context, tokenStr string) (accessToken, refreshToken string, err error) {
	claims, err := s.ParseToken(tokenStr, jwt.WithExpirationRequired())
	if err != nil || claims.Type != refreshTokenType {
		return "", "", ErrInvalidToken
	}

	valid, err := s.refreshRepo.IsValid(ctx, claims.ID)
	if err != nil {
		return "", "", fmt.Errorf("check token validity: %w", err)
	}
	if !valid {
		_ = s.refreshRepo.RevokeAllForUser(ctx, claims.Subject)
		return "", "", ErrTokenRevoked
	}

	if err := s.refreshRepo.MarkRevoked(ctx, claims.ID); err != nil {
		return "", "", fmt.Errorf("revoke old token: %w", err)
	}

	return s.GenerateTokenPair(ctx, claims.Subject, claims.Role)
}

func (s *service) RevokeSession(ctx context.Context, accessTokenStr, refreshTokenStr string) error {
	accessClaims, err := s.ParseToken(accessTokenStr, jwt.WithExpirationRequired())
	if err != nil {
		return ErrInvalidToken
	}

	refreshClaims, err := s.ParseToken(refreshTokenStr)
	if err != nil {
		return ErrInvalidToken
	}

	if err := s.refreshRepo.MarkRevoked(ctx, refreshClaims.ID); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if ttl := time.Until(accessClaims.ExpiresAt.Time); ttl > 0 {
		if err := s.redis.Set(ctx, blacklistKey(accessClaims.ID), "1", ttl).Err(); err != nil {
			return fmt.Errorf("blacklist access token: %w", err)
		}
	}
	return nil
}

func (s *service) ParseToken(tokenStr string, opts ...jwt.ParserOption) (*Claims, error) {
	parseopts := make([]jwt.ParserOption, 0, 2+len(opts))
	parseopts = append(parseopts, jwt.WithIssuer(s.issuer), jwt.WithAudience(s.audience))
	parseopts = append(parseopts, opts...)
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, s.keyFunc(), parseopts...)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (s *service) keyFunc() jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	}
}

func (s *service) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	err := s.redis.Get(ctx, blacklistKey(jti)).Err()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func blacklistKey(jti string) string {
	return "jwt:blacklist:" + jti
}
