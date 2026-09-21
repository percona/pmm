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

package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestServiceInfoRequestValkeyForwardsTLSSettings(t *testing.T) {
	t.Parallel()
	f := newValkeyTLSRequestFixture(t)

	request, err := serviceInfoRequest(f.db.Querier, f.service, f.agent)
	require.NoError(t, err)

	assert.True(t, request.Tls)
	assert.True(t, request.TlsSkipVerify)
	assert.Contains(t, request.Dsn, "rediss://")
	assert.Equal(t, map[string]string{"tlsCa": "ca-pem"}, request.TextFiles.Files)
	require.NoError(t, f.mock.ExpectationsWereMet())
}

// Skip-verify is the one setting that has to survive with no TLS material at all, since
// that is exactly the self-signed case it exists for.
func TestServiceInfoRequestValkeyForwardsSkipVerifyWithoutCertificates(t *testing.T) {
	t.Parallel()
	f := newValkeyTLSRequestFixture(t)
	f.agent.ValkeyOptions = models.ValkeyOptions{}

	request, err := serviceInfoRequest(f.db.Querier, f.service, f.agent)
	require.NoError(t, err)

	assert.True(t, request.Tls)
	assert.True(t, request.TlsSkipVerify)
	assert.Empty(t, request.TextFiles.GetFiles())
	require.NoError(t, f.mock.ExpectationsWereMet())
}
