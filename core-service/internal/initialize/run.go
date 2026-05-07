package initialize

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info().Msg("shutdown signal received")
		cancel()
	}()

	teamHandler, assetHandler, jwks := initServices(ctx, cfg, dbPool, rdb)

	r := router.New(jwks, rdb, teamHandler, assetHandler)
	log.Info().Str("port", cfg.Port).Msg("starting core-service")
	return r.Run(":" + cfg.Port)
}
