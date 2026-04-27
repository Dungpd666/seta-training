package initialize

import (
	"context"

	"github.com/dungpd/seta/core-service/internal/config"
	"github.com/dungpd/seta/core-service/internal/db"
	"github.com/dungpd/seta/core-service/internal/middleware"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func initServices(cfg *config.Config, dbPool *pgxpool.Pool, rdb *redis.Client) (*team.Handler, *middleware.JWKSClient) {
	q := db.New(dbPool)

	projectionRepo := team.NewProjectionRepository(q)
	StartUserEventConsumer(context.Background(), cfg.KafkaBrokers, projectionRepo)
	teamRepo := team.NewTeamRepository(q)
	teamSvc := team.NewService(teamRepo)
	teamHandler := team.NewHandler(teamSvc)

	jwks := middleware.NewJWKSClient(cfg.JWKSUrl)

	return teamHandler, jwks
}
