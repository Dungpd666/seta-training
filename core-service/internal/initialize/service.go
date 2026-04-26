package initialize

import (
	"github.com/dungpd/seta/core-service/internal/db"
	"github.com/dungpd/seta/core-service/internal/team"
	"github.com/jackc/pgx/v5/pgxpool"
)

func initProjectionRepo(dbPool *pgxpool.Pool) team.ProjectionRepository {
	q := db.New(dbPool)
	return team.NewProjectionRepository(q)
}
