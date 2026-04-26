package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DBURL          string
	Port           string
	MigrationsPath string
	RedisURL       string
	PublicKey      *rsa.PublicKey
	KafkaBrokers   []string
}

func Load() (*Config, error) {
	dbURL, err := require("DB_URL")
	if err != nil {
		return nil, err
	}

	publicKey, err := loadPublicKey()
	if err != nil {
		return nil, fmt.Errorf("public key: %w", err)
	}

	return &Config{
		DBURL:          dbURL,
		Port:           getenv("PORT", "8082"),
		MigrationsPath: getenv("MIGRATIONS_PATH", "migrations"),
		RedisURL:       getenv("REDIS_URL", "redis://localhost:6379/0"),
		PublicKey:      publicKey,
		KafkaBrokers:   strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ","),
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
