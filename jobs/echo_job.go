// pmm-agent
// Copyright 2019 Percona LLC
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

package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/sirupsen/logrus"
)

type echoJob struct {
	id      string
	timeout time.Duration
	l       *logrus.Entry

	message string
	delay   time.Duration
}

// NewEchoJob create simple echo job for testing purposes.
func NewEchoJob(id string, timeout time.Duration, message string, delay time.Duration) Job {
	return &echoJob{
		id:      id,
		timeout: timeout,
		l:       logrus.WithFields(logrus.Fields{"id": id, "type": "echo"}),
		message: message,
		delay:   delay,
	}
}

// ID returns job id.
func (j *echoJob) ID() string {
	return j.id
}

// Type returns job type.
func (j *echoJob) Type() string {
	return "echo"
}

// Timeouts returns job timeout.
func (j *echoJob) Timeout() time.Duration {
	return j.timeout
}

// Run runs job.
func (j *echoJob) Run(ctx context.Context, send Send) error {
	j.l.Info("Job started.")
	send(&agentpb.JobProgress{
		JobId:     j.id,
		Timestamp: ptypes.TimestampNow(),
		Result: &agentpb.JobProgress_Echo_{
			Echo: &agentpb.JobProgress_Echo{
				Status: fmt.Sprintf("Echo job %s started.", j.id),
			}}})
	delay := time.NewTimer(j.delay)
	defer delay.Stop()

	select {
	case <-delay.C:
		send(&agentpb.JobResult{
			JobId:     j.id,
			Timestamp: ptypes.TimestampNow(),
			Result: &agentpb.JobResult_Echo_{
				Echo: &agentpb.JobResult_Echo{
					Message: j.message,
				}}})
	case <-ctx.Done():
		return ctx.Err()
	}

	j.l.Info("Job complete")
	return nil
}
