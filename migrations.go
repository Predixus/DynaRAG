package dynarag

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed internal/store/migrations/*.sql
var migrations embed.FS

// initMigrations runs database migrations using embedded migration files
func initMigrations(postgresConnStr string) error {
	d, err := iofs.New(migrations, "store/migrations")
	if err != nil {
		return fmt.Errorf("failed to load embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, postgresConnStr)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
