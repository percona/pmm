// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package managedapi describes how a co-located process reaches pmm-managed's HTTP API.
//
// The values live here because neither has an importable owner: the listen address is a
// package-level var in pmm-managed's main, and the route is bound by generated gateway code to
// an unexported pattern rather than to a string.
package managedapi

const (
	// HTTPPort is the port pmm-managed serves its own HTTP API on.
	HTTPPort = "7772"

	// LeaderHealthCheckPath reports whether this node holds cluster leadership: 200 on the
	// leader, and FailedPrecondition rendered as 400 everywhere else. It needs no
	// credentials, see the exemption in managed/services/grafana/auth_server.go.
	LeaderHealthCheckPath = "/v1/server/leaderHealthCheck"
)
