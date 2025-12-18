package migrate

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed sql/*.sql
var migrationsFS embed.FS

// Apply runs all migrations up using the embedded migration files.
func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	srcDriver, err := iofs.New(migrationsFS, "sql")
	if err != nil {
		return fmt.Errorf("init iofs: %w", err)
	}

	sqlDB, err := sql.Open("pgx", pool.Config().ConnString())
	if err != nil {
		return fmt.Errorf("open sql db: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sql db: %w", err)
	}

	dbDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("init db driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", srcDriver, "pgx", dbDriver)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("migrate up: %w (hint: ensure every migration version has both `.up.sql` and `.down.sql`, and rebuild the Docker image since migrations are embedded in the `migrate` binary)", err)
		}
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
