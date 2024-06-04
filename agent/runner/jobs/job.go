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

// Package jobs provides Jobs implementations and runner.
package jobs

import (
	"context"
	"time"

	"github.com/percona/pmm/api/agentpb"
)

// JobType represents Job type.
type JobType string

// Available job types.
const (
	MySQLBackup    = JobType("mysql_backup")
	MongoDBBackup  = JobType("mongodb_backup")
	MongoDBRestore = JobType("mongodb_restore")
	MySQLRestore   = JobType("mysql_restore")
)

// Send is interface for function that used by jobs to send messages back to pmm-server.
type Send func(payload agentpb.AgentResponsePayload)

// Job represents job interface.
type Job interface {
	// ID returns Job ID.
	ID() string
	// Type returns Job type.
	Type() JobType
	// Timeout returns Job timeout.
	Timeout() time.Duration
	// DSN returns Data Source Name required for the Action.
	DSN() string
	// Run starts Job execution.
	Run(ctx context.Context, send Send) error
}
