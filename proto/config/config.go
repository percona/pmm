/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package config

type Agent struct {
	UUID        string
	ApiHostname string
	ApiPath     string
	Keepalive   uint              `json:",omitempty"`
	PidFile     string            `json:",omitempty"`
	Links       map[string]string `json:",omitempty"`
}

type Data struct {
	Encoding     string `json:",omitempty"`
	SendInterval uint   `json:",omitempty"`
	Blackhole    string `json:",omitempty"` // dev
	Limits       DataSpoolLimits
}

type DataSpoolLimits struct {
	MaxAge   uint   // seconds
	MaxSize  uint64 // bytes
	MaxFiles uint
}

type Log struct {
	Level   string `json:",omitempty"`
	Offline string `json:",omitempty"` // dev
}

type QAN struct {
	UUID                    string  // of MySQL instance
	CollectFrom             string  // "slowlog" or "perfschema"
	Interval                uint    // seconds, 0 = DEFAULT_INTERVAL
	LongQueryTime           float64 // microseconds, 0 or >= 0.000001
	MaxSlowLogSize          int64   // bytes, 0 = DEFAULT_MAX_SLOW_LOG_SIZE
	RemoveOldSlowLogs       bool    // after rotating for MaxSlowLogSize
	ExampleQueries          bool    // send real example of each query
	SlowLogVerbosity        string  // "minimal", "standard", "full"
	RateLimit               uint    // Percona Server query sampling
	LogSlowAdminStatements  bool    // Percona Server feature
	LogSlowSlaveStatemtents bool    // Percona Server feature
	// internal
	Start         []string // queries to configure MySQL (enable slow log, etc.)
	Stop          []string // queries to un-configure MySQL (disable slow log, etc.)
	WorkerRunTime uint     // seconds, 0 = DEFAULT_WORKER_RUNTIME
	ReportLimit   uint     // top N queries, 0 = DEFAULT_REPORT_LIMIT
}

// Response for GET /qan/:uuid/config
type RunningQAN struct {
	AgentUUID     string
	SetConfig     string `json:",omitempty"`
	RunningConfig string `json:",omitempty"`
}
