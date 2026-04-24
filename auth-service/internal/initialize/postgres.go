package initialize

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

func Database(dbURL string) (*pgxpool.Pool, error) {
	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	if err := dbPool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("database ping: %w", err)
	}
	log.Info().Msg("connected to database")
	return dbPool, nil
}

func Migrations(dbURL, path string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.Up(db, path); err != nil {
		return err
	}
	log.Info().Msg("migrations applied")
	return nil
}
