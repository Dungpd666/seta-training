package initialize

import (
	"context"

	"github.com/dungpd/seta/core-service/internal/asset"
	"github.com/dungpd/seta/core-service/internal/config"
	"github.com/dungpd/seta/core-service/internal/db"
	"github.com/dungpd/seta/core-service/internal/middleware"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func initServices(cfg *config.Config, dbPool *pgxpool.Pool, rdb *redis.Client) (*team.Handler, *asset.Handler, *middleware.JWKSClient) {
	q := db.New(dbPool)

	projectionRepo := team.NewProjectionRepository(q)
	StartUserEventConsumer(context.Background(), cfg.KafkaBrokers, projectionRepo)
	teamRepo := team.NewTeamRepository(q)
	teamSvc := team.NewTeamService(teamRepo)
	teamHandler := team.NewTeamHandler(teamSvc)

	assetRepo := asset.NewRepository(q)
	assetSvc := asset.NewService(assetRepo)
	assetHandler := asset.NewHandler(assetSvc)

	jwks := middleware.NewJWKSClient(cfg.JWKSUrl)

	return teamHandler, assetHandler, jwks
}
