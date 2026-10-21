// Copyright (C) 2023 Percona LLC
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

package dump

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPMMDumpEnv(t *testing.T) {
	t.Parallel()

	env := pmmDumpEnv(
		[]string{"PATH=/usr/bin", "PMM_DEBUG=1", "PMM_TOKEN=inherited", "PMM_DUMP_PASS=inherited", "PMM_PASS=inherited"},
		map[string]string{
			"PMM_DUMP_CLICK_HOUSE_URL": "tcp://default:s3cr3t@127.0.0.1:9000/pmm",
			"PMM_TOKEN":                "",
			"PMM_DUMP_PASS":            "dump-pass",
		},
	)
	assert.Equal(t, []string{
		"PATH=/usr/bin",
		"PMM_DEBUG=1",
		"PMM_DUMP_CLICK_HOUSE_URL=tcp://default:s3cr3t@127.0.0.1:9000/pmm",
		"PMM_DUMP_PASS=dump-pass",
	}, env, "the variables of pmm-dump are its caller's only")
}
