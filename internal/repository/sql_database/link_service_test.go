package repository_test

import (
	"context"
	"fmt"
	repository "go-progira/internal/repository/sql_database"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startTestPostgres(t *testing.T) (string, error) {
	t.Helper()

	ctx := context.Background()

	postgresPort := "5432"
	postgresUser := "test"
	postgresPassword := "test"
	postgresDB := "testdb"

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{postgresPort + "/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     postgresUser,
			"POSTGRES_PASSWORD": postgresPassword,
			"POSTGRES_DB":       postgresDB,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(10 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", err
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			slog.Error("failed to terminate container",
				slog.String("error", err.Error()))
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", err
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", postgresUser, postgresPassword, host, port.Port(), postgresDB)

	return dbURL, nil
}

func TestNewLinkService(t *testing.T) {
	dbURL, _ := startTestPostgres(t)

	t.Run("returns ORMLinkService", func(t *testing.T) {
		svc, err := repository.NewLinkService("orm", dbURL)
		assert.NoError(t, err)
		assert.IsType(t, &repository.ORMLinkService{}, svc)
	})

	t.Run("returns SQLLinkService", func(t *testing.T) {
		svc, err := repository.NewLinkService("sql", dbURL)
		assert.NoError(t, err)
		assert.IsType(t, &repository.SQLLinkService{}, svc)
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		svc, err := repository.NewLinkService("invalid", dbURL)
		assert.Error(t, err)
		assert.Nil(t, svc)
	})
}

func TestCreateChat(t *testing.T) {
	ctx := context.Background()

	dbURL, err := startTestPostgres(t)
	require.NoError(t, err)

	db, err := pgxpool.Connect(ctx, dbURL)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			telegram_id BIGINT PRIMARY KEY
		)
	`)
	require.NoError(t, err)

	tests := []struct {
		name string
		typ  string
	}{
		{"SQL implementation", "sql"},
		{"ORM implementation", "orm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := repository.NewLinkService(tt.typ, dbURL)
			require.NoError(t, err)

			db, err := pgxpool.Connect(ctx, dbURL)
			require.NoError(t, err)
			defer db.Close()

			telegramID := int64(1234567)
			err = svc.CreateChat(ctx, telegramID)
			assert.NoError(t, err)

			var count int
			err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE telegram_id = $1", telegramID).Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)

			err = svc.CreateChat(ctx, telegramID)
			assert.NoError(t, err)

			err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE telegram_id = $1", telegramID).Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	}
}

func TestDeleteChat(t *testing.T) {
	ctx := context.Background()

	dbURL, err := startTestPostgres(t)
	require.NoError(t, err)

	db, err := pgxpool.Connect(ctx, dbURL)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			telegram_id BIGINT PRIMARY KEY
		)
	`)

	db.Close()
	require.NoError(t, err)

	tests := []struct {
		name string
		typ  string
	}{
		{"SQL implementation", "sql"},
		{"ORM implementation", "orm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := repository.NewLinkService(tt.typ, dbURL)
			require.NoError(t, err)

			db, err := pgxpool.Connect(ctx, dbURL)
			require.NoError(t, err)
			defer db.Close()

			ID := int64(1234567)
			_, err = db.Exec(ctx, "INSERT INTO users (telegram_id) VALUES ($1) ON CONFLICT (telegram_id) DO NOTHING", ID)
			assert.NoError(t, err)

			err = svc.DeleteChat(ctx, ID)
			assert.NoError(t, err)

			var count int
			err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE telegram_id = $1", ID).Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 0, count)

			err = svc.DeleteChat(ctx, ID)
			assert.NoError(t, err)
		})
	}
}

func TestAddLink(t *testing.T) {
	ctx := context.Background()
	dbURL, err := startTestPostgres(t)
	require.NoError(t, err)

	db, err := pgxpool.Connect(ctx, dbURL)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL
	);
	CREATE TABLE IF NOT EXISTS links (
		id SERIAL PRIMARY KEY,
		url TEXT UNIQUE NOT NULL,
		changed_at TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS link_users (
		user_id INT REFERENCES users(id),
		link_id INT REFERENCES links(id),
		UNIQUE(user_id, link_id)
	);
	CREATE TABLE IF NOT EXISTS tags (
		id SERIAL PRIMARY KEY,
		name TEXT UNIQUE NOT NULL
	);
	CREATE TABLE IF NOT EXISTS link_tags (
		link_id INT REFERENCES links(id),
		tag_id INT REFERENCES tags(id),
		UNIQUE(link_id, tag_id)
	);
	CREATE TABLE IF NOT EXISTS filters (
		id SERIAL PRIMARY KEY,
		name TEXT UNIQUE NOT NULL
	);
	CREATE TABLE IF NOT EXISTS link_filters (
		link_id INT REFERENCES links(id),
		filter_id INT REFERENCES filters(id),
		UNIQUE(link_id, filter_id)
	);`)

	db.Close()

	require.NoError(t, err)

	tests := []struct {
		name string
		typ  string
	}{
		{"SQL implementation", "sql"},
		{"ORM implementation", "orm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := repository.NewLinkService(tt.typ, dbURL)
			require.NoError(t, err)

			db, err := pgxpool.Connect(ctx, dbURL)
			require.NoError(t, err)
			defer db.Close()

			telegramID := int64(1111)
			_, err = db.Exec(ctx, `INSERT INTO users (telegram_id) VALUES ($1) ON CONFLICT DO NOTHING`, telegramID)
			require.NoError(t, err)

			url := "https://example.com"
			tags := []string{"tag1", "tag2"}
			filters := []string{"filter1", "filter2"}

			err = svc.AddLink(ctx, telegramID, url, tags, filters)
			require.NoError(t, err)

			var linkID int
			err = db.QueryRow(ctx, `SELECT id FROM links WHERE url = $1`, url).Scan(&linkID)
			assert.NoError(t, err)
			assert.Greater(t, linkID, 0)

			var count int
			err = db.QueryRow(ctx, `
				SELECT COUNT(*) FROM link_users
				WHERE link_id = $1 AND user_id = (SELECT id FROM users WHERE telegram_id = $2)
			`, linkID, telegramID).Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)

			for _, tag := range tags {
				var tagID int
				err := db.QueryRow(ctx, `SELECT id FROM tags WHERE name = $1`, tag).Scan(&tagID)
				assert.NoError(t, err)

				err = db.QueryRow(ctx, `
					SELECT COUNT(*) FROM link_tags WHERE link_id = $1 AND tag_id = $2
					`, linkID, tagID).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			}

			for _, filter := range filters {
				var filterID int
				err := db.QueryRow(ctx, `SELECT id FROM filters WHERE name = $1`, filter).Scan(&filterID)
				assert.NoError(t, err)

				err = db.QueryRow(ctx, `
					SELECT COUNT(*) FROM link_filters WHERE link_id = $1 AND filter_id = $2
					`, linkID, filterID).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)
			}
		})
	}
}

func TestSaveLastUpdate(t *testing.T) {
	ctx := context.Background()

	dbURL, err := startTestPostgres(t)
	require.NoError(t, err)

	db, err := pgxpool.Connect(ctx, dbURL)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS links (
		id SERIAL PRIMARY KEY,
		url TEXT UNIQUE NOT NULL,
		changed_at TIMESTAMP
		)
	`)

	db.Close()
	require.NoError(t, err)

	tests := []struct {
		name string
		typ  string
	}{
		{"SQL implementation", "sql"},
		{"ORM implementation", "orm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := repository.NewLinkService(tt.typ, dbURL)
			require.NoError(t, err)

			db, err := pgxpool.Connect(ctx, dbURL)
			require.NoError(t, err)
			defer db.Close()

			url := "https://example.com"

			_, err = db.Exec(ctx,
				"INSERT INTO links (url, changed_at) VALUES ($1, NOW()) ON CONFLICT DO NOTHING", url)
			require.NoError(t, err)

			var linkID int64
			err = db.QueryRow(ctx, "SELECT id FROM links WHERE url = $1", url).Scan(&linkID)
			require.NoError(t, err)

			expectedUpdTime := time.Date(2025, time.May, 5, 18, 24, 0, 0, time.UTC)

			err = svc.SaveLastUpdate(ctx, linkID, expectedUpdTime)
			assert.NoError(t, err)

			var actualUpdTime time.Time

			err = db.QueryRow(ctx,
				"SELECT changed_at FROM links WHERE id = $1", linkID).Scan(&actualUpdTime)
			assert.NoError(t, err)

			assert.Equal(t, actualUpdTime, expectedUpdTime)
		})
	}
}
