package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dungpd/seta/auth-service/internal/config"
	"github.com/dungpd/seta/auth-service/internal/handler"
	"github.com/dungpd/seta/auth-service/internal/middleware"
	"github.com/dungpd/seta/auth-service/internal/repository"
	"github.com/dungpd/seta/auth-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if err := run(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := runMigrations(cfg.DBURL, cfg.MigrationsPath); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DBURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	log.Info().Msg("connected to database")

	rdb, err := connectRedis(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	log.Info().Msg("connected to redis")

	userRepo := repository.NewUserRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	userSvc := service.NewUserService(userRepo)
	authSvc := service.NewAuthService(refreshRepo, cfg.PrivateKey, cfg.PublicKey, rdb)
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

	protected := r.Group("/")
	protected.Use(middleware.JWTAuth(authSvc))
	protected.GET("/users", h.ListUsers)

	log.Info().Str("port", cfg.Port).Msg("starting auth-service")
	return r.Run(":" + cfg.Port)
}

func connectRedis(url string) (*redis.Client, error) {
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

func runMigrations(dbURL, path string) error {
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
