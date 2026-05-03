package db

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func RunMigrations(ctx context.Context, conn *sql.DB, migrationsDir string) error {
	if err := conn.PingContext(ctx); err != nil {
		return err
	}
	goose.SetDialect("postgres")
	return goose.UpContext(ctx, conn, migrationsDir)
}
