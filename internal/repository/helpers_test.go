package repository

import (
	"database/sql"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/stretchr/testify/require"
)

func upDatabase(t *testing.T) *sql.DB {
	conn, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	err = conn.Ping()
	require.NoError(t, err)

	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "sqlite3", driver)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)

	return conn
}