package migrations

import (
	"io"
	"sort"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/sirupsen/logrus"
)

type memMigration struct {
	Version    uint
	Identifier string
	Up         string
	Down       string
}

type memMigrations []memMigration

func newMemMigrations(migs []memMigration) memMigrations {
	sort.Slice(migs, func(i, j int) bool {
		return migs[i].Version < migs[j].Version
	})

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
		if s[i].Version == version {
			if i+1 < len(s) {
				return s[i+1].Version, nil
			}
			// No next migration, return 0 and io.EOF
			return 0, io.EOF
		}
	}

	return 0, io.EOF
}

func (s memMigrations) ReadUp(version uint) (io.ReadCloser, string, error) {
	for _, m := range s {
		if m.Version == version {
			logrus.Debugf("[memMigrations] ReadUp: version=%d, identifier=%s\nSQL:\n%s\n", m.Version, m.Identifier, m.Up)

			return io.NopCloser(strings.NewReader(m.Up)), m.Identifier, nil
		}
	}

	return nil, "", io.EOF
}

func (s memMigrations) ReadDown(version uint) (io.ReadCloser, string, error) {
	for _, m := range s {
		if m.Version == version && m.Down != "" {
			return io.NopCloser(strings.NewReader(m.Down)), m.Identifier, nil
		}
	}

	return nil, "", io.EOF
}
func (s memMigrations) Reset() error            { return nil }
func (s memMigrations) Name() string            { return "memMigrations" }
func (s memMigrations) Lock() error             { return nil }
func (s memMigrations) Unlock() error           { return nil }
func (s memMigrations) LastModified() time.Time { return time.Now() }
