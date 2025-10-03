package migrations

import (
	"io"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/sirupsen/logrus"
)

type memMigration struct {
	Identifier string
	Up         string
	Down       string
}

type memMigrations struct {
	migs     []memMigration
	versions []uint
}

func newMemMigrations(migs []memMigration, versions []uint) memMigrations {
	return memMigrations{migs: migs, versions: versions}
}

func (s memMigrations) Open(url string) (source.Driver, error) { return s, nil }
func (s memMigrations) Close() error                           { return nil }
func (s memMigrations) Reset() error                           { return nil }
func (s memMigrations) Name() string                           { return "memMigrations" }
func (s memMigrations) Lock() error                            { return nil }
func (s memMigrations) Unlock() error                          { return nil }
func (s memMigrations) LastModified() time.Time                { return time.Now() }

func (s memMigrations) ReadUp(version uint) (io.ReadCloser, string, error) {
	for i, v := range s.versions {
		if v == version {
			m := s.migs[i]
			logrus.Debugf("[memMigrations] ReadUp: version=%d, identifier=%s\nSQL:\n%s\n", v, m.Identifier, m.Up)

			return io.NopCloser(strings.NewReader(m.Up)), m.Identifier, nil
		}
	}

	return nil, "", io.EOF
}

func (s memMigrations) ReadDown(version uint) (io.ReadCloser, string, error) {
	for i, v := range s.versions {
		if v == version && s.migs[i].Down != "" {
			m := s.migs[i]
			return io.NopCloser(strings.NewReader(m.Down)), m.Identifier, nil
		}
	}

	return nil, "", io.EOF
}

func (s memMigrations) First() (uint, error) {
	if len(s.versions) == 0 {
		return 0, io.EOF
	}
	return s.versions[0], nil
}

func (s memMigrations) Prev(version uint) (uint, error) {
	for i, v := range s.versions {
		if v == version && i > 0 {
			return s.versions[i-1], nil
		}
	}
	return 0, io.EOF
}

func (s memMigrations) Next(version uint) (uint, error) {
	for i, v := range s.versions {
		if v == version {
			if i+1 < len(s.versions) {
				return s.versions[i+1], nil
			}
			return 0, io.EOF
		}
	}
	return 0, io.EOF
}
