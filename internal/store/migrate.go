package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"

	"github.com/codepuke/codepuke/migrations"
)

// RunMigrations applies every pending migration. goose needs database/sql,
// so this opens its own short-lived handle; the server's pgxpool is created
// separately after migrations succeed.
func RunMigrations(ctx context.Context, databaseURL string) (err error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer func() {
		err = errors.Join(err, db.Close())
	}()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
