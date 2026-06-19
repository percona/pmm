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

package realtimeanalytics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestIsRtaFeatureSupported(t *testing.T) {
	t.Parallel()

	// MongoDB RTA shipped in 3.7.0.
	assert.True(t, isRtaFeatureSupported("3.7.0", models.MongoDBServiceType))
	assert.True(t, isRtaFeatureSupported("3.8.0", models.MongoDBServiceType))
	assert.False(t, isRtaFeatureSupported("3.6.0", models.MongoDBServiceType))

	// MySQL RTA shipped in 3.9.0 — an agent in [3.7.0, 3.9.0) supports MongoDB RTA but
	// would not understand the MySQL builtin, so it must be reported as unsupported.
	assert.False(t, isRtaFeatureSupported("3.7.0", models.MySQLServiceType))
	assert.False(t, isRtaFeatureSupported("3.8.0", models.MySQLServiceType))
	assert.False(t, isRtaFeatureSupported("3.8.99", models.MySQLServiceType))
	assert.True(t, isRtaFeatureSupported("3.9.0", models.MySQLServiceType))
	assert.True(t, isRtaFeatureSupported("3.10.0", models.MySQLServiceType))

	// Service types that do not support RTA are never reported as supported,
	// regardless of agent version.
	assert.False(t, isRtaFeatureSupported("3.9.0", models.ValkeyServiceType))
	assert.False(t, isRtaFeatureSupported("3.9.0", models.PostgreSQLServiceType))

	// Unparsable version is never supported.
	assert.False(t, isRtaFeatureSupported("not-a-version", models.MySQLServiceType))
}
