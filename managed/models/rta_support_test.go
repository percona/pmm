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

package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/models"
)

func TestIsRTASupported(t *testing.T) {
	t.Parallel()

	// MongoDB RTA shipped in 3.7.0.
	assert.True(t, models.IsRTASupported("3.7.0", models.MongoDBServiceType))
	assert.True(t, models.IsRTASupported("3.8.0", models.MongoDBServiceType))
	assert.False(t, models.IsRTASupported("3.6.0", models.MongoDBServiceType))

	// MySQL RTA ships in 3.10.0 — an agent in [3.7.0, 3.10.0) supports MongoDB RTA but
	// would not understand the MySQL builtin, so it must be reported as unsupported.
	// 3.9.0 and 3.9.1 were released without the MySQL collector.
	assert.False(t, models.IsRTASupported("3.7.0", models.MySQLServiceType))
	assert.False(t, models.IsRTASupported("3.8.0", models.MySQLServiceType))
	assert.False(t, models.IsRTASupported("3.9.0", models.MySQLServiceType))
	assert.False(t, models.IsRTASupported("3.9.1", models.MySQLServiceType))
	assert.True(t, models.IsRTASupported("3.10.0", models.MySQLServiceType))
	assert.True(t, models.IsRTASupported("3.10.1", models.MySQLServiceType))

	// PostgreSQL RTA ships in 3.10.0 as well.
	assert.False(t, models.IsRTASupported("3.9.1", models.PostgreSQLServiceType))
	assert.True(t, models.IsRTASupported("3.10.0", models.PostgreSQLServiceType))

	// Service types that do not support RTA are never reported as supported,
	// regardless of agent version.
	assert.False(t, models.IsRTASupported("3.9.0", models.ValkeyServiceType))

	// Unparsable version is never supported.
	assert.False(t, models.IsRTASupported("not-a-version", models.MySQLServiceType))
}

func TestRTANotSupportedError(t *testing.T) {
	t.Parallel()

	err := models.RTANotSupportedError("mysql80-oldclient-391", "svc-1", "3.9.1", models.MySQLServiceType)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Equal(t, "Service mysql80-oldclient-391 (id svc-1) has pmm-agent with version 3.9.1 not supporting Real-Time Analytics; "+
		"pmm-agent 3.10.0 or later is required.", status.Convert(err).Message())

	err = models.RTANotSupportedError("mongo-1", "svc-2", "", models.MongoDBServiceType)
	assert.Equal(t, "Service mongo-1 (id svc-2) has pmm-agent with version unknown not supporting Real-Time Analytics; "+
		"pmm-agent 3.7.0 or later is required.", status.Convert(err).Message())

	err = models.RTANotSupportedError("valkey-1", "svc-3", "3.10.0", models.ValkeyServiceType)
	assert.Equal(t, "Service valkey-1 (id svc-3) of type valkey does not support Real-Time Analytics.", status.Convert(err).Message())
}
