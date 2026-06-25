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

package realtimeanalytics

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

type lockRow struct {
	BlockedPID     int32
	BlockerPID     int32
	LockMode       string
	RelationName   string
	BlockerQuery   string
	BlockerQueryAt time.Time
}

type sessionRow struct {
	PID                  int32
	DatabaseName         string
	Username             string
	ApplicationName      string
	ClientAddress        string
	State                string
	Query                string
	QueryID              int64
	HasQueryID           bool
	BackendStart         time.Time
	TransactionStart     time.Time
	QueryStart           time.Time
	WaitEventType        string
	WaitEvent            string
	LeaderPID            int32
	TrackActivitySize    int32
	QueryTextTruncated   bool
	RawJSON              string
}

func buildLockChains(rows []lockRow, sessions map[int32]*sessionRow) map[int32][]*rtav1.LockChainLink {
	chains := make(map[int32][]*rtav1.LockChainLink)
	now := time.Now()

	for _, row := range rows {
		link := &rtav1.LockChainLink{
			BlockerPid:       row.BlockerPID,
			BlockedPid:       row.BlockedPID,
			LockMode:         row.LockMode,
			RelationName:     row.RelationName,
			BlockerQueryText: row.BlockerQuery,
		}

		if blocker, ok := sessions[row.BlockerPID]; ok && !blocker.QueryStart.IsZero() {
			link.BlockerDuration = durationpb.New(now.Sub(blocker.QueryStart))
		} else if !row.BlockerQueryAt.IsZero() {
			link.BlockerDuration = durationpb.New(now.Sub(row.BlockerQueryAt))
		}

		chains[row.BlockedPID] = append(chains[row.BlockedPID], link)
	}

	return chains
}
