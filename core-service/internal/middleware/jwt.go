package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dungpd/seta/core-service/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	CtxUserID       = "user_id"
	CtxRole         = "role"
	accessTokenType = "access"
)

type Claims struct {
	Role string `json:"role"`
	Type string `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

type JWKSClient struct {
	url      string
	mu       sync.RWMutex
	keys     map[string]*rsa.PublicKey
	issuer   string
	audience string
	http     *http.Client
}

func NewJWKSClient(jwksURL, issuer, audience string) *JWKSClient {
	return &JWKSClient{
		url:      jwksURL,
		keys:     make(map[string]*rsa.PublicKey),
		issuer:   issuer,
		audience: audience,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *JWKSClient) GetKey(kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if key, ok := c.keys[kid]; ok {
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	return c.refresh(kid)
}

func (c *JWKSClient) refresh(kid string) (*rsa.PublicKey, error) {
	resp, err := c.http.Get(c.url)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, k := range jwks.Keys {
		key, err := jwkToRSA(k.N, k.E)
		if err != nil {
			continue
		}
		c.keys[k.Kid] = key
	}

	key, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("kid %q not found in jwks", kid)
	}
	return key, nil
}

func jwkToRSA(n, e string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

func JWTAuth(jwks *JWKSClient, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "missing or invalid Authorization header")
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &Claims{}
		_, err := jwt.ParseWithClaims(tokenStr, claims,
			func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				kid, _ := t.Header["kid"].(string)
				return jwks.GetKey(kid)
			},
			jwt.WithIssuer(jwks.issuer),
			jwt.WithAudience(jwks.audience),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "invalid token")
			return
		}

		if claims.Type != "" && claims.Type != accessTokenType {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "invalid token type")
			return
		}

		blacklisted, err := rdb.Exists(c.Request.Context(), "jwt:blacklist:"+claims.ID).Result()
		if err != nil {
			log.Error().Err(err).Msg("redis blacklist check failed")
		} else if blacklisted > 0 {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "token revoked")
			return
		}

		c.Set(CtxUserID, claims.Subject)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}
