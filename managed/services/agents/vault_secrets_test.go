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
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
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
func TestChangedServerSecrets(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	stored := mapSecrets{"secret/data/pmm#agent_password": "first"}
	secrets.Configure(stored, time.Minute)
	t.Cleanup(func() { secrets.Configure(nil, 0) })

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "vault-node", Address: "127.0.0.1"})
	require.NoError(t, err)
	pmmAgent, err := models.CreatePMMAgent(db.Querier, node.NodeID, nil)
	require.NoError(t, err)
	_, err = models.CreateNodeExporter(db.Querier, pmmAgent.AgentID, nil, false, false, nil, new("vault:secret/data/pmm#agent_password"), "")
	require.NoError(t, err)

	u := &StateUpdater{db: db}
	l := logrus.WithField("test", t.Name())
	seen := make(map[string]string)

	assert.Empty(t, u.changedServerSecrets(t.Context(), seen, l), "the first check only takes note")
	assert.Len(t, seen, 1)
	assert.Empty(t, u.changedServerSecrets(t.Context(), seen, l))

	stored["secret/data/pmm#agent_password"] = "rotated"
	assert.Equal(t, map[string]struct{}{pmmAgent.AgentID: {}}, u.changedServerSecrets(t.Context(), seen, l))
	assert.Empty(t, u.changedServerSecrets(t.Context(), seen, l))

	// An unreadable secret is not a change: the Agents keep running with what they have.
	delete(stored, "secret/data/pmm#agent_password")
	assert.Empty(t, u.changedServerSecrets(t.Context(), seen, l))
}

// Not parallel: it configures the process-wide Vault client of PMM Server.
func TestJobS3KeysFromVault(t *testing.T) {
	t.Cleanup(func() { secrets.Configure(nil, 0) })
	location := &models.S3LocationConfig{
		Endpoint: "https://s3.example.com", BucketName: "b",
		AccessKey: "vault:secret/data/s3#access_key", SecretKey: "vault:secret/data/s3#secret_key",
	}

	_, err := convertS3ConfigModel(location)
	require.ErrorContains(t, err, "Failed to read the keys of the backup location: cannot read vault:secret/data/s3#access_key: PMM Server has no Vault configured")

	secrets.Configure(mapSecrets{"secret/data/s3#access_key": "AKIA", "secret/data/s3#secret_key": "s3cr3t"}, time.Minute)
	cfg, err := convertS3ConfigModel(location)
	require.NoError(t, err)
	assert.Equal(t, "AKIA", cfg.AccessKey)
	assert.Equal(t, "s3cr3t", cfg.SecretKey)
	assert.Equal(t, "vault:secret/data/s3#secret_key", location.SecretKey, "the location itself is not changed")
}

type fakeVictoriaMetrics struct {
	forced chan struct{}
}

func (f *fakeVictoriaMetrics) RequestConfigurationUpdate() {}

func (f *fakeVictoriaMetrics) ForceConfigurationUpdate(context.Context) error {
	f.forced <- struct{}{}
	return nil
}

func (f *fakeVictoriaMetrics) BuildScrapeConfigForVMAgent(context.Context, string) ([]byte, error) {
	return nil, nil
}

// Not parallel: it configures the process-wide Vault client of PMM Server.
func TestWatchServerSecretsUpdatesVictoriaMetricsAtOnce(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	stored := &lockedSecrets{m: map[string]string{"secret/data/pmm#agent_password": "first"}}
	secrets.Configure(stored, time.Minute)
	t.Cleanup(func() { secrets.Configure(nil, 0) })

	_, err := models.CreateNodeExporter(db.Querier, models.PMMServerAgentID, nil, false, false, nil, new("vault:secret/data/pmm#agent_password"), "")
	require.NoError(t, err)

	vmdb := &fakeVictoriaMetrics{forced: make(chan struct{}, 1)}
	u := &StateUpdater{db: db, r: &Registry{}, vmdb: vmdb}
	ctx, cancel := context.WithCancel(logger.Set(t.Context(), t.Name()))
	done := make(chan struct{})
	go func() {
		defer close(done)
		u.WatchServerSecrets(ctx, 10*time.Millisecond)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	select {
	case <-vmdb.forced:
		t.Fatal("updated without a change")
	case <-time.After(100 * time.Millisecond):
	}

	stored.set("secret/data/pmm#agent_password", "rotated")
	select {
	case <-vmdb.forced:
	case <-time.After(5 * time.Second):
		t.Fatal("not updated after the secret changed")
	}
}

type lockedSecrets struct {
	mu sync.Mutex
	m  map[string]string
}

func (s *lockedSecrets) Get(_ context.Context, path, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.m[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

func (s *lockedSecrets) set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m[k] = v
}
