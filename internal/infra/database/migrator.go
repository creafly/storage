package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Migrator struct {
	db            *sqlx.DB
	migrationsDir string
}

func NewMigrator(db *sqlx.DB, migrationsDir string) *Migrator {
	return &Migrator{db: db, migrationsDir: migrationsDir}
}

func (m *Migrator) Up() error {
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	files, err := m.getMigrationFiles("up")
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	for _, file := range files {
		version := m.extractVersion(file)
		if m.isMigrationApplied(version) {
			continue
		}

		log.Printf("Applying migration: %s", file)
		if err := m.executeMigration(file); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		if err := m.recordMigration(version); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}
	}

	return nil
}

func (m *Migrator) Down() error {
	files, err := m.getMigrationFiles("down")
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	if len(files) == 0 {
		return nil
	}

	file := files[0]
	version := m.extractVersion(file)

	if !m.isMigrationApplied(version) {
		return nil
	}

	log.Printf("Rolling back migration: %s", file)
	if err := m.executeMigration(file); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", file, err)
	}

	if err := m.removeMigration(version); err != nil {
		return fmt.Errorf("failed to remove migration record %s: %w", version, err)
	}

	return nil
}

func (m *Migrator) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) getMigrationFiles(direction string) ([]string, error) {
	pattern := filepath.Join(m.migrationsDir, fmt.Sprintf("*.%s.sql", direction))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (m *Migrator) extractVersion(filename string) string {
	base := filepath.Base(filename)
	parts := strings.Split(base, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return base
}

func (m *Migrator) isMigrationApplied(version string) bool {
	var count int
	_ = m.db.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version)
	return count > 0
}

func (m *Migrator) executeMigration(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(string(content))
	return err
}

func (m *Migrator) recordMigration(version string) error {
	_, err := m.db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
	return err
}

func (m *Migrator) removeMigration(version string) error {
	_, err := m.db.Exec("DELETE FROM schema_migrations WHERE version = $1", version)
	return err
}
