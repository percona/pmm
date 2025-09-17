package main

import (
	"io"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4/source"
)

type memMigration struct {
	Version    uint
	Identifier string
	Up         string
}

type dynamicMigrations struct {
	migrations []memMigration
}

func newDynamicMigrations(migrations []memMigration) *dynamicMigrations {
	return &dynamicMigrations{migrations: migrations}
}

func (s *dynamicMigrations) Open(url string) (source.Driver, error) { return s, nil }
func (s *dynamicMigrations) Close() error                           { return nil }
func (s *dynamicMigrations) First() (uint, error) {
	if len(s.migrations) == 0 {
		return 0, io.EOF
	}

	return s.migrations[0].Version, nil
}

func (s *dynamicMigrations) Prev(version uint) (uint, error) {
	for i := range s.migrations {
		if s.migrations[i].Version == version && i > 0 {
			return s.migrations[i-1].Version, nil
		}
	}

	return 0, io.EOF
}

func (s *dynamicMigrations) Next(version uint) (uint, error) {
	for i := range s.migrations {
		if s.migrations[i].Version == version && i+1 < len(s.migrations) {
			return s.migrations[i+1].Version, nil
		}
	}

	return 0, io.EOF
}

func (s *dynamicMigrations) ReadUp(version uint) (io.ReadCloser, string, error) {
	for _, m := range s.migrations {
		if m.Version == version {
			return io.NopCloser(strings.NewReader(m.Up)), m.Identifier, nil
		}
	}

	return nil, "", io.EOF
}

func (s *dynamicMigrations) ReadDown(version uint) (io.ReadCloser, string, error) {
	return nil, "", io.EOF
}
func (s *dynamicMigrations) Reset() error            { return nil }
func (s *dynamicMigrations) Name() string            { return "dynamic" }
func (s *dynamicMigrations) Lock() error             { return nil }
func (s *dynamicMigrations) Unlock() error           { return nil }
func (s *dynamicMigrations) LastModified() time.Time { return time.Now() }
