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
package grafana

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	stringsgen "github.com/percona/pmm/utils/strings"
)

func Test_sanitizeSAName(t *testing.T) {
	// max possible length without hashing
	len180, err := stringsgen.GenerateRandomString(180)
	require.NoError(t, err)
	require.Equal(t, len180, SanitizeSAName(len180))

	// too long length - postfix hashed
	len200, err := stringsgen.GenerateRandomString(200)
	require.NoError(t, err)
	len200sanitized := SanitizeSAName(len200)
	require.Equal(t, fmt.Sprintf("%s%s", len200[:148], len200sanitized[148:]), len200sanitized)
}
