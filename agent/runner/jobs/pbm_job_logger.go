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

package jobs

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/api/agentpb"
)

type pbmJob string

const (
	pbmBackupJob  pbmJob = "backup"
	pbmRestoreJob pbmJob = "restore"
)

type pbmJobLogger struct {
	dbURL      string
	jobID      string
	jobType    pbmJob
	logChunkID uint32
}

func newPbmJobLogger(jobID string, jobType pbmJob, mongoURL string) *pbmJobLogger {
	return &pbmJobLogger{
		jobID:      jobID,
		jobType:    jobType,
		logChunkID: 0,
		dbURL:      mongoURL,
	}
}

func (l *pbmJobLogger) sendLog(send Send, data string, done bool) {
	send(&agentpb.JobProgress{
		JobId:     l.jobID,
		Timestamp: timestamppb.Now(),
		Result: &agentpb.JobProgress_Logs_{
			Logs: &agentpb.JobProgress_Logs{
				ChunkId: atomic.AddUint32(&l.logChunkID, 1) - 1,
				Data:    data,
				Done:    done,
			},
		},
	})
}

func (l *pbmJobLogger) streamLogs(ctx context.Context, send Send, name string) error {
	var (
		err    error
		logs   []pbmLogEntry
		buffer bytes.Buffer
		skip   int
	)
	l.logChunkID = 0

	ticker := time.NewTicker(logsCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logs, err = retrieveLogs(ctx, l.dbURL, fmt.Sprintf("%s/%s", l.jobType, name))
			if err != nil {
				return err
			}
			// @TODO Replace skip with proper paging after this is done https://jira.percona.com/browse/PBM-713
			logs = logs[skip:]
			skip += len(logs)
			if len(logs) == 0 {
				continue
			}
			from, to := 0, maxLogsChunkSize
			for from < len(logs) {
				if to > len(logs) {
					to = len(logs)
				}
				buffer.Reset()
				for i, log := range logs[from:to] {
					_, err := buffer.WriteString(log.String())
					if err != nil {
						return err
					}
					if i != to-from-1 {
						buffer.WriteRune('\n')
					}
				}
				l.sendLog(send, buffer.String(), false)
				from += maxLogsChunkSize
				to += maxLogsChunkSize
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
