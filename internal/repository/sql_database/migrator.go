package repository

import (
	"database/sql"
	"embed"
	"errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"log"
)

const migrationsDir = "migrations"

//go:embed migrations/*.sql
var sqlFiles embed.FS

type Migrator struct {
	srcDriver source.Driver
	fs        embed.FS
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

	migrator, err := migrate.NewWithInstance("migration_embeded_sql_files", m.srcDriver, "psql_db", driver)
	if err != nil {
		log.Printf("unable to create migration: %v", err)
		return err
	}

	defer func() {
		migrator.Close()
	}()

	if err = migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Printf("unable to apply migrations %v", err)
		return err
	}

	return nil
}

//func IsMigrationsApplied(db *sql.DB) bool {
//	var exists bool
//	query := `
//        SELECT EXISTS (
//            SELECT 1
//            FROM   information_schema.tables
//            WHERE  table_schema = 'public'
//            AND    table_name = 'schema_migrations'
//        );
//    `
//	_ = db.QueryRow(query).Scan(&exists)
//	return exists
//}
