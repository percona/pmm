// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sender

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/agents/mongodb/profiler/internal/report"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

type testWriter struct {
	t              *testing.T
	expectedReport *report.Report
}

func (w *testWriter) Write(actual *report.Report) error {
	assert.NotNil(w.t, actual)
	assert.Equal(w.t, w.expectedReport, actual)
	return nil
}

func TestSender(t *testing.T) {
	expected := &report.Report{
		StartTS: time.Now(),
		EndTS:   time.Now().Add(time.Second * 10),
		Buckets: []*agentv1.MetricsBucket{{Common: &agentv1.MetricsBucket_Common{Queryid: "test"}}},
	}

	repChan := make(chan *report.Report)
	tw := &testWriter{t: t, expectedReport: expected}
	snd := New(repChan, tw, logrus.WithField("component", "test-sender"))
	err := snd.Start()
	require.NoError(t, err)

	repChan <- expected
	snd.Stop()
}
