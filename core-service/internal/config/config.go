package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DBURL          string
	Port           string
	MigrationsPath string
	RedisURL       string
	KafkaBrokers   []string
	JWKSUrl        string
}

func Load() (*Config, error) {
	dbURL, err := require("DB_URL")
	if err != nil {
		return nil, err
	}

	return &Config{
		DBURL:          dbURL,
		Port:           getenv("PORT", "8082"),
		MigrationsPath: getenv("MIGRATIONS_PATH", "migrations"),
		RedisURL:       getenv("REDIS_URL", "redis://localhost:6379/0"),
		KafkaBrokers:   strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ","),
		JWKSUrl:        getenv("JWKS_URL", "http://localhost:8081/.well-known/jwks.json"),
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

