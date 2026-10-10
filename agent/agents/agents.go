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

// Package agents contains PMM agents implementations.
package agents

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

// RTAQueryTag marks the queries the MySQL Real-Time Analytics agent polls with, so the QAN agents
// can leave them out: RTA runs them every collect interval on the very server QAN is watching.
// Each query selects NULL AS pmm_agent_rta as its first column, because the statement digest that
// the perfschema QAN agent reads keeps identifiers but strips comments, and is cut at
// max_digest_length (1024 bytes by default) -- these queries are longer than that, so the tag has
// to come first. The /* pmm-agent:rta */ comment next to it is for whoever reads the processlist
// or the slow log. It lives here because agents must not import each other.
const RTAQueryTag = "pmm_agent_rta"

// IsRTAQuery reports whether a query text, or its digest text, is one of the MySQL Real-Time
// Analytics agent's polling queries.
func IsRTAQuery(query string) bool {
	return strings.Contains(query, RTAQueryTag)
}

// Change represents built-in Agent status change and/or QAN collect request.
type Change struct {
	Status           inventoryv1.AgentStatus
	MetricsBucket    []*agentv1.MetricsBucket
	RTAQueriesBucket []*rtav1.QueryData

	// StatusMessage explains Status to the user, e.g. why initialization failed. Optional.
	StatusMessage string
}

// BuiltinAgent is a common interface for all built-in Agents.
type BuiltinAgent interface {
	// Run extracts stats data and sends it to the channel until ctx is canceled.
	Run(ctx context.Context)

	// Changes returns channel that should be read until it is closed.
	Changes() <-chan Change

	// Collector added to use BuiltinAgent as Prometheus collector
	prometheus.Collector
}
