package testutil

import (
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/hexaend/storage/internal/domain/entity"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type TestDB struct {
	DB *sqlx.DB
}

func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	dbURL := os.Getenv("TEST_STORAGE_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5440/storage_test?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	runMigrations(t, db)
	cleanupTables(t, db)

	return &TestDB{DB: db}
}

func runMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../../../migrations",
		"postgres", driver)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to run migrations: %v", err)
	}
}

func cleanupTables(t *testing.T, db *sqlx.DB) {
	t.Helper()

	tables := []string{"files"}
	for _, table := range tables {
		_, err := db.Exec("DELETE FROM " + table)
		if err != nil {
			t.Fatalf("Failed to clean table %s: %v", table, err)
		}
	}
}

func (tdb *TestDB) Cleanup(t *testing.T) {
	t.Helper()
	cleanupTables(t, tdb.DB)
	tdb.DB.Close()
}

func NewTestFile() *entity.File {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &entity.File{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		UploadedBy:   uuid.New(),
		FileName:     "test-file-" + uuid.New().String()[:8] + ".png",
		OriginalName: "original-image.png",
		ContentType:  "image/png",
		FileType:     entity.FileTypeImage,
		Size:         1024,
		Path:         "/tenants/test/images/test.png",
		URL:          "https://storage.example.com/tenants/test/images/test.png",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func NewTestFileWithTenant(tenantID uuid.UUID) *entity.File {
	file := NewTestFile()
	file.TenantID = tenantID
	return file
}

func NewTestFileWithType(fileType entity.FileType) *entity.File {
	file := NewTestFile()
	file.FileType = fileType
	return file
}
