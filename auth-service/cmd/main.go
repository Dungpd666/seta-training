package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/dungpd/seta/auth-service/internal/handler"
	"github.com/dungpd/seta/auth-service/internal/repository"
	"github.com/dungpd/seta/auth-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if err := run(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run() error {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Warn().Msg("DB_URL not set, using insecure default — do not use in production")
		dbURL = "postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable"
	}

	if err := runMigrations(dbURL); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	log.Info().Msg("connected to database")

	privateKey, err := loadPrivateKey()
	if err != nil {
		return fmt.Errorf("private key: %w", err)
	}
	publicKey, err := loadPublicKey()
	if err != nil {
		return fmt.Errorf("public key: %w", err)
	}
	rdb, err := connectRedis()
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	userRepo := repository.NewUserRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(refreshRepo, privateKey, publicKey, rdb)
	h := handler.NewUserHandler(userSvc, authSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Msg("request")
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(503, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/.well-known/jwks.json", h.JWKS)
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Info().Str("port", port).Msg("starting auth-service")
	return r.Run(":" + port)
}

func runMigrations(dbURL string) error {
	path := os.Getenv("MIGRATIONS_PATH")
	if path == "" {
		path = "migrations"
	}
	m, err := migrate.New("file://"+path, dbURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	log.Info().Msg("migrations applied")
	return nil
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
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
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
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
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
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	log.Info().Msg("connected to Redis")
	return rdb, nil
}
