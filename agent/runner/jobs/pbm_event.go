package jobs

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/api/agentpb"
)

type pbmEvent string

const (
	pbmBackupEvent  pbmEvent = "backup"
	pbmRestoreEvent pbmEvent = "restore"
)

type pbmEventLog struct {
	dbURL       *url.URL
	eventType   pbmEvent
	parentJobID string
	logChunkID  uint32
}

func newPbmEventLog(parentJobId string, eventType pbmEvent, mongoUrl *url.URL) *pbmEventLog {
	return &pbmEventLog{
		parentJobID: parentJobId,
		eventType:   eventType,
		logChunkID:  0,
		dbURL:       mongoUrl,
	}
}

func (l *pbmEventLog) sendLog(send Send, data string, done bool) {
	send(&agentpb.JobProgress{
		JobId:     l.parentJobID,
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

func (l *pbmEventLog) streamLogs(ctx context.Context, send Send, eventType pbmEvent, eventName string) error {
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
			logs, err = retrieveLogs(ctx, l.dbURL, fmt.Sprintf("%s/%s", eventType, eventName))
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
