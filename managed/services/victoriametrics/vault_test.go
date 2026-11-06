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

package victoriametrics

import (
	"context"
	"errors"
	"testing"
	"time"

	config "github.com/percona/promconfig"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/managed/utils/testdb"
)

type mapSecrets map[string]string

func (m mapSecrets) Get(_ context.Context, path, key string) (string, error) {
	v, ok := m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

// Not parallel: it configures the process-wide Vault client of PMM Server.
func TestScrapeConfigsWithSecretsFromVault(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	stored := mapSecrets{"secret/data/pmm#agent_password": "metrics-pw"}
	secrets.Configure(stored, time.Minute)
	t.Cleanup(func() { secrets.Configure(nil, 0) })

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "vault-node", Address: "1.2.3.4"})
	require.NoError(t, err)
	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)
	exporter, err := models.CreateNodeExporter(db.Querier, pmmAgent.AgentID, nil, false, false, nil, new("vault:secret/data/pmm#agent_password"), "")
	require.NoError(t, err)
	exporter.ListenPort = new(uint16(42000))
	require.NoError(t, models.UpdateAgent(db.Querier, exporter))

	resolutions := &models.MetricsResolutions{HR: time.Second, MR: 5 * time.Second, LR: time.Minute}
	scrape := func() []*config.ScrapeConfig {
		cfg := &config.Config{}
		require.NoError(t, AddScrapeConfigs(logrus.WithField("test", t.Name()), cfg, db.Querier, resolutions, nil, false, false))
		return cfg.ScrapeConfigs
	}

	configs := scrape()
	require.NotEmpty(t, configs)
	for _, c := range configs {
		require.NotNil(t, c.HTTPClientConfig.BasicAuth, c.JobName)
		assert.Equal(t, "metrics-pw", c.HTTPClientConfig.BasicAuth.Password, c.JobName)
	}

	// An exporter whose password PMM Server cannot read is left out, rather than scraped with the reference.
	delete(stored, "secret/data/pmm#agent_password")
	assert.Empty(t, scrape())
}
