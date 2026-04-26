package initialize

import (
	"context"
	"fmt"

	"github.com/dungpd/seta/core-service/internal/config"
	"github.com/dungpd/seta/core-service/internal/router"
	"github.com/rs/zerolog/log"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := Migrations(cfg.DBURL, cfg.MigrationsPath); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	dbPool, err := Database(cfg.DBURL)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	_, err = Redis(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	log.Info().Msg("connected to redis")

	projectionRepo := initProjectionRepo(dbPool)
	StartUserEventConsumer(context.Background(), cfg.KafkaBrokers, projectionRepo)

	r := router.New()
	log.Info().Str("port", cfg.Port).Msg("starting core-service")
	return r.Run(":" + cfg.Port)
}
