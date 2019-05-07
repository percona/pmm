// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package channel

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "pmm_managed"
	prometheusSubsystem = "channel"
)

// SharedChannelMetrics represents channel metrics shared between all channels.
type SharedChannelMetrics struct {
	mRecv prom.Counter
	mSend prom.Counter
}

// NewSharedMetrics creates new SharedChannelMetrics.
func NewSharedMetrics() *SharedChannelMetrics {
	return &SharedChannelMetrics{
		mRecv: prom.NewCounter(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "messages_received_total",
			Help:      "A total number of messages received from pmm-agents.",
		}),
		mSend: prom.NewCounter(prom.CounterOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "messages_sent_total",
			Help:      "A total number of messages sent to pmm-agents.",
		}),
	}
}

// Describe implements prometheus.Collector.
func (scm *SharedChannelMetrics) Describe(ch chan<- *prom.Desc) {
	scm.mRecv.Describe(ch)
	scm.mSend.Describe(ch)
}

// Collect implement prometheus.Collector.
func (scm *SharedChannelMetrics) Collect(ch chan<- prom.Metric) {
	scm.mRecv.Collect(ch)
	scm.mSend.Collect(ch)

	// TODO metrics for channel's len(requests) and cap(requests)
}

// check interfaces
var (
	_ prom.Collector = (*SharedChannelMetrics)(nil)
)
