package repository

import (
	"database/sql"
	"log"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"embed"
	"errors"
)

const migrationsDir = "migrations"

//go:embed migrations/*.sql
var sqlFiles embed.FS

type Migrator struct {
	srcDriver source.Driver
}

func MustGetNewMigrator() *Migrator {
	d, err := iofs.New(sqlFiles, migrationsDir)
	if err != nil {
		panic(err)
	}

	return &Migrator{
		srcDriver: d,
	}
}

func (m *Migrator) ApplyMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Printf("unable to create db instance: %v", err)
		return err
	}

	migrator, err := migrate.NewWithInstance("migration_embedded_sql_files", m.srcDriver,
		"pg_db", driver)
	if err != nil {
		log.Printf("unable to create migration: %v", err)
		return err
	}

	errCloseSource, errCloseDB := migrator.Close()
	if errCloseSource != nil {
		slog.Error("Error closing source")

		return errCloseSource
	}

	if errCloseDB != nil {
		slog.Error("Error closing database")

		return errCloseDB
	}

	if err = migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Printf("unable to apply migrations %v", err)
		return err
	}

	return nil
}
