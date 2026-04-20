package config

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Config struct {
	DBURL      string
	Port       string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Redis      *redis.Client
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Warn().Msg("DB_URL not set, using insecure default — do not use in production")
		dbURL = "postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	privateKey, err := loadPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}

	publicKey, err := loadPublicKey()
	if err != nil {
		return nil, fmt.Errorf("public key: %w", err)
	}

	rdb, err := connectRedis()
	if err != nil {
		return nil, fmt.Errorf("redis: %w", err)
	}

	return &Config{
		DBURL:      dbURL,
		Port:       port,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Redis:      rdb,
	}, nil
}

func loadPrivateKey() (*rsa.PrivateKey, error) {
	path := os.Getenv("JWT_PRIVATE_KEY_PATH")
	if path == "" {
		path = "jwt_rs256.pem"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w — generate with: openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out jwt_rs256.pem", path, err)
	}
	block, _ := pem.Decode(data)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA key")
	}
	return rsaKey, nil
}

func loadPublicKey() (*rsa.PublicKey, error) {
	path := os.Getenv("JWT_PUBLIC_KEY_PATH")
	if path == "" {
		path = "jwt_rs256.pub"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w — generate with: openssl rsa -in jwt_rs256.pem -pubout -out jwt_rs256.pub", path, err)
	}
	block, _ := pem.Decode(data)
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA key")
	}
	return rsaKey, nil
}

func connectRedis() (*redis.Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/0"
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return rdb, nil
}
