package initialize

import (
	"github.com/dungpd/seta/auth-service/internal/auth"
	"github.com/dungpd/seta/auth-service/internal/config"
	"github.com/dungpd/seta/auth-service/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func initServices(
	cfg *config.Config,
	dbPool *pgxpool.Pool,
	rdb *redis.Client,
) (*auth.Handler, *user.Handler, auth.Service) {
	userRepo := user.NewRepository(dbPool)
	refreshRepo := auth.NewRefreshTokenRepository(dbPool)
	userSvc := user.NewService(
		userRepo,
		user.WithWorkers(cfg.ImportWorkers),
	)
	authSvc := auth.NewService(refreshRepo, cfg.PrivateKey, cfg.PublicKey, rdb)
	return auth.NewHandler(userSvc, authSvc), user.NewHandler(userSvc), authSvc
}
