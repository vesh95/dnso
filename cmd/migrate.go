/*
Copyright © 2026 Eduard Larionov <vesh95.17@ya.ru>
*/
package cmd

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/spf13/cobra"
)

var MigrationsFS embed.FS

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMigrateUp(); err != nil {
			log.Fatalf("migrate up error: %v", err)
		}
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback all migrations",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMigrateDown(); err != nil {
			log.Fatalf("migrate down error: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
}

func dbPath() string {
	if val := os.Getenv("DNSO_DB_PATH"); val != "" {
		return val
	}
	return "./dnso.db"
}

func newMigrate() (*migrate.Migrate, error) {
	dbURL := fmt.Sprintf("sqlite3://%s", dbPath())

	src, err := iofs.New(MigrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	return m, nil
}

func runMigrateUp() error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("migrate up failed: %w", err)
	}

	log.Println("Migrations applied successfully")
	return nil
}

func runMigrateDown() error {
	m, err := newMigrate()
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Down(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No migrations to rollback")
			return nil
		}
		return fmt.Errorf("migrate down failed: %w", err)
	}

	log.Println("Migrations rolled back successfully")
	return nil
}
