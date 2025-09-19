package migrations

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

type memMigrations []memMigration

func newMemMigrations(migs []memMigration) memMigrations {
	return memMigrations(migs)
}

func (s memMigrations) Open(url string) (source.Driver, error) { return s, nil }
func (s memMigrations) Close() error                           { return nil }
func (s memMigrations) First() (uint, error) {
	if len(s) == 0 {
		return 0, io.EOF
	}

	return s[0].Version, nil
}

func (s memMigrations) Prev(version uint) (uint, error) {
	for i := range s {
		if s[i].Version == version && i > 0 {
			return s[i-1].Version, nil
		}
	}

	return 0, io.EOF
}

func (s memMigrations) Next(version uint) (uint, error) {
	for i := range s {
		if s[i].Version == version && i+1 < len(s) {
			return s[i+1].Version, nil
		}
	}

	return 0, io.EOF
}

func (s memMigrations) ReadUp(version uint) (io.ReadCloser, string, error) {
	for _, m := range s {
		if m.Version == version {
			return io.NopCloser(strings.NewReader(m.Up)), m.Identifier, nil
		}
	}
	return nil, "", io.EOF
}

func (s memMigrations) ReadDown(version uint) (io.ReadCloser, string, error) {
	return nil, "", io.EOF
}
func (s memMigrations) Reset() error            { return nil }
func (s memMigrations) Name() string            { return "memMigrations" }
func (s memMigrations) Lock() error             { return nil }
func (s memMigrations) Unlock() error           { return nil }
func (s memMigrations) LastModified() time.Time { return time.Now() }
