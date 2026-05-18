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

func initServices(ctx context.Context, cfg *config.Config, dbPool *pgxpool.Pool, rdb *redis.Client) (*team.Handler, *asset.Handler, *middleware.JWKSClient) {
	q := db.New(dbPool)
	producer := NewKafkaProducer(cfg.KafkaBrokers)

	projectionRepo := team.NewProjectionRepository(q)
	StartUserEventConsumer(ctx, cfg.KafkaBrokers, projectionRepo)
	StartAuditConsumer(ctx, cfg.KafkaBrokers, q)

	teamRepo := team.NewRepository(q, dbPool)
	teamSvc := team.NewService(teamRepo, rdb, producer)
	teamHandler := team.NewHandler(teamSvc)

	assetRepo := asset.NewRepository(q)
	assetSvc := asset.NewService(assetRepo, rdb, producer)
	assetHandler := asset.NewHandler(assetSvc)

	jwks := middleware.NewJWKSClient(cfg.JWKSUrl, cfg.JWTIssuer, cfg.JWTAudience)
	return teamHandler, assetHandler, jwks
}
