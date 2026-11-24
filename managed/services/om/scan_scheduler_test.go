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

package om

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

type fakeClusterScanRequester struct {
	clusters []string
	profile  string
}

func (f *fakeClusterScanRequester) RequestClusterScan(_ context.Context, clusterID, profile, _ string) error {
	f.clusters = append(f.clusters, clusterID)
	f.profile = profile
	return nil
}

func clusterWithStatus(t *testing.T, q *reform.Querier, status models.OmClusterStatus) string {
	t.Helper()
	c := &models.OmCluster{
		ID: uuid.New().String(), Name: "rs0", Kind: models.OmClusterReplicaSet,
		Origin: models.OmClusterInstalled, Status: status, CreatedBy: "t",
	}
	require.NoError(t, q.Insert(c))
	return c.ID
}

func TestWithScanSchedulerComposition(t *testing.T) {
	db := lockTestDB(t)
	l := logrus.WithField("test", t.Name())

	off := New(db, nil, nil, l).WithScanScheduler("", "")
	assert.Nil(t, off.scanScheduler, "an empty Extensions URL leaves managed scans off")
	off.RunScanScheduler(t.Context())

	on := New(db, nil, nil, l).WithScanScheduler("http://extensions.test", "token")
	require.NotNil(t, on.scanScheduler, "a configured Extensions URL composes the scheduler")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	on.RunScanScheduler(ctx)
}

func TestScanSchedulerRequestsManagedClustersOnly(t *testing.T) {
	db := lockTestDB(t)
	managedA := clusterWithStatus(t, db.Querier, models.OmClusterManaged)
	managedB := clusterWithStatus(t, db.Querier, models.OmClusterManaged)
	clusterWithStatus(t, db.Querier, models.OmClusterMonitored)

	fake := &fakeClusterScanRequester{}
	scheduler := NewScanScheduler(db, logrus.WithField("test", t.Name()), fake)
	scheduler.Tick(t.Context())

	assert.ElementsMatch(t, []string{managedA, managedB}, fake.clusters)
	assert.Equal(t, scanProfileManaged, fake.profile)
}

func TestScanSchedulerNonLeaderDoesNothing(t *testing.T) {
	db := lockTestDB(t)
	clusterWithStatus(t, db.Querier, models.OmClusterManaged)

	fake := &fakeClusterScanRequester{}
	scheduler := NewScanScheduler(db, logrus.WithField("test", t.Name()), fake).
		WithLeader(fakeHA{leader: false, leaderID: "node-b"})
	scheduler.Tick(t.Context())

	assert.Empty(t, fake.clusters)
}
