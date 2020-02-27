// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package supervisord

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	pmmUpdateCheck := NewPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker_logs"))
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	s := New(configDir, pmmUpdateCheck)
	settings := &models.Settings{
		DataRetention: 30 * 24 * time.Hour,
	}

	for _, tmpl := range templates.Templates() {
		if tmpl.Name() == "" {
			continue
		}

		tmpl := tmpl
		t.Run(tmpl.Name(), func(t *testing.T) {
			expected, err := ioutil.ReadFile(filepath.Join(configDir, tmpl.Name()+".ini")) //nolint:gosec
			require.NoError(t, err)
			actual, err := s.marshalConfig(tmpl, settings)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}

func TestParseStatus(t *testing.T) {
	t.Parallel()

	for str, expected := range map[string]*bool{
		`pmm-agent                        STOPPED   Sep 20 08:55 AM`:         pointer.ToBool(false),
		`pmm-managed                      RUNNING   pid 826, uptime 0:19:36`: pointer.ToBool(true),
		`pmm-update-perform               EXITED    Sep 20 07:42 AM`:         nil,
		`pmm-update-perform               STARTING`:                          pointer.ToBool(true), // no last column in that case
	} {
		assert.Equal(t, expected, parseStatus(str), "%q", str)
	}
}
