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

package agentlocal

import (
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/prometheus/client_golang/prometheus"
)

//go:generate mockery -name=client -case=snake -inpkg -testonly

// client is a subset of methods of client.Client used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type client interface {
	GetServerConnectMetadata() *agentpb.ServerConnectMetadata
	Describe(chan<- *prometheus.Desc)
	Collect(chan<- prometheus.Metric)
	GetNetworkInformation() (latency, clockDrift time.Duration, err error)
}
