package initialize

import (
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

	rdb, err := Redis(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	log.Info().Msg("connected to redis")

	teamHandler, jwks := initServices(cfg, dbPool, rdb)

	r := router.New(jwks, rdb, teamHandler)
	log.Info().Str("port", cfg.Port).Msg("starting core-service")
	return r.Run(":" + cfg.Port)
}
