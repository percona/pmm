// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package profiler

import (
	"testing"
	"time"

	"github.com/percona/pmgo"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/mgo.v2/bson"

	"github.com/percona/pmm-agent/agents/builtin/mongodb/internal/profiler/aggregator"
	"github.com/percona/pmm-agent/agents/builtin/mongodb/internal/report"
)

func TestProfiler(t *testing.T) {
	defaultInterval := aggregator.DefaultInterval
	aggregator.DefaultInterval = time.Duration(time.Second)
	defer func() { aggregator.DefaultInterval = defaultInterval }()

	url := "mongodb://root:root-password@127.0.0.1:27017"

	dialInfo, err := pmgo.ParseURL(url)
	require.NoError(t, err)

	dialer := pmgo.NewDialer()

	sess, err := createSession(dialInfo, dialer)
	require.NoError(t, err)

	err = sess.DB("test").DropDatabase()
	require.NoError(t, err)

	ms := &testWriter{t: t}
	prof := New(dialInfo, dialer, logrus.WithField("component", "profiler-test"), ms, "test-id")
	err = prof.Start()
	require.NoError(t, err)

	err = sess.DB("test").C("peoples").Insert(bson.M{"name": "Anton"}, bson.M{"name": "Alexey"})
	assert.NoError(t, err)

	<-time.After(aggregator.DefaultInterval)

	err = prof.Stop()
	require.NoError(t, err)
}

type testWriter struct {
	t *testing.T
}

func (tw *testWriter) Write(actual *report.Report) error {
	require.NotNil(tw.t, actual)
	assert.Equal(tw.t, 1, len(actual.Buckets))

	expected := &qanpb.MetricsBucket{
		Fingerprint:        "INSERT peoples",
		Database:           "test",
		Schema:             "peoples",
		AgentId:            "test-id",
		AgentType:          inventorypb.AgentType_QAN_MONGODB_PROFILER_AGENT,
		NumQueries:         1,
		MResponseLengthSum: 60,
		MResponseLengthMin: 60,
		MResponseLengthMax: 60,
	}

	assert.Equal(tw.t, expected, actual.Buckets[0])
	return nil
}
