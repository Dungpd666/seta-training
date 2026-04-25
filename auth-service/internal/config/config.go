package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DBURL          string
	Port           string
	MigrationsPath string
	RedisURL       string
	PrivateKey     *rsa.PrivateKey
	PublicKey      *rsa.PublicKey
	ImportWorkers  int
}

func Load() (*Config, error) {
	dbURL, err := require("DB_URL")
	if err != nil {
		return nil, err
	}

	privateKey, err := loadPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}

	publicKey, err := loadPublicKey()
	if err != nil {
		return nil, fmt.Errorf("public key: %w", err)
	}

	return &Config{
		DBURL:          dbURL,
		Port:           getenv("PORT", "8081"),
		MigrationsPath: getenv("MIGRATIONS_PATH", "migrations"),
		RedisURL:       getenv("REDIS_URL", "redis://localhost:6379/0"),
		PrivateKey:     privateKey,
		PublicKey:      publicKey,
		ImportWorkers:  getenvInt("IMPORT_WORKERS", 5),
	}, nil
}

func require(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required env var %s is not set", key)
	}
	return v, nil
}

func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getenvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

func loadPrivateKey() (*rsa.PrivateKey, error) {
	path := getenv("JWT_PRIVATE_KEY_PATH", "jwt_rs256.pem")
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
	path := getenv("JWT_PUBLIC_KEY_PATH", "jwt_rs256.pub")
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
