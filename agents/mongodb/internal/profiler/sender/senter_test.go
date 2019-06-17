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

package sender

import (
	"testing"
	"time"

	"github.com/percona/pmm/api/qanpb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/agents/mongodb/internal/report"
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
		StartTs: time.Now(),
		EndTs:   time.Now().Add(time.Second * 10),
		Buckets: []*qanpb.MetricsBucket{{Queryid: "test"}},
	}

	repChan := make(chan *report.Report)
	tw := &testWriter{t: t, expectedReport: expected}
	snd := New(repChan, tw, logrus.WithField("component", "test-sender"))
	err := snd.Start()
	require.NoError(t, err)

	repChan <- expected
	snd.Stop()
}
