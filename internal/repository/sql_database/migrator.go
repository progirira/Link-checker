package repository

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
)

type Migrator struct {
	MigrationsPath string
}

func (m *Migrator) ApplyMigrations(db *sql.DB) error {
	if _, err := os.Stat(m.MigrationsPath); os.IsNotExist(err) {
		slog.Error("Migrations directory doesn't exist",
			slog.String("error", err.Error()))

		return err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("Failed to create db driver",
			slog.String("error", err.Error()))

		return err
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://"+m.MigrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		slog.Error("failed to create migrator",
			slog.String("error", err.Error()))
	}

	defer func() {
		if _, errClose := migrator.Close(); errClose != nil {
			slog.Error("migration close error",
				slog.String("error", errClose.Error()))

			err = errClose
		}
	}()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("migration failed",
			slog.String("error", err.Error()))
	}

	return err
}
