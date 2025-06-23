package services

import (
	"database/sql"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pkg/errors"
)

// MigrationService handles database migrations using go-migrate with embedded files
type MigrationService struct {
	db         *sql.DB
	embeddedFS fs.FS
}

// NewMigrationService creates a new migration service with embedded migrations
func NewMigrationService(db *sql.DB, embeddedFS fs.FS) *MigrationService {
	return &MigrationService{
		db:         db,
		embeddedFS: embeddedFS,
	}
}

// createMigrateInstance creates a migrate instance using embedded filesystem
func (s *MigrationService) createMigrateInstance() (*migrate.Migrate, error) {
	// Create postgres driver instance
	driver, err := postgres.WithInstance(s.db, &postgres.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create postgres driver")
	}

	// Create embedded filesystem source driver
	sourceDriver, err := iofs.New(s.embeddedFS, "migrations")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create iofs source driver")
	}

	// Create migrate instance with embedded migrations
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create migrate instance")
	}

	return m, nil
}

// RunMigrations runs all pending database migrations
func (s *MigrationService) RunMigrations() error {
	m, err := s.createMigrateInstance()
	if err != nil {
		return err
	}
	defer m.Close()

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "failed to run migrations")
	}

	return nil
}

// GetMigrationVersion returns the current migration version
func (s *MigrationService) GetMigrationVersion() (uint, bool, error) {
	m, err := s.createMigrateInstance()
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	// Get current version
	version, dirty, err := m.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil
		}
		return 0, false, errors.Wrap(err, "failed to get migration version")
	}

	return version, dirty, nil
}
