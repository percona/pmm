// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package slackbot

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	slackEventsTotal = prom.NewCounter(prom.CounterOpts{
		Name: "pmm_slack_adre_events_total",
		Help: "Slack events handled by the ADRE Slack integration.",
	})
	adreChatSeconds = prom.NewHistogram(prom.HistogramOpts{
		Name:    "pmm_slack_adre_chat_seconds",
		Help:    "Latency of ADRE chat requests initiated from the Slack integration.",
		Buckets: prom.ExponentialBuckets(0.5, 2, 14), //nolint:mnd
	})
	slackUploadsTotal = prom.NewCounter(prom.CounterOpts{
		Name: "pmm_slack_adre_image_uploads_total",
		Help: "Panel PNG images uploaded to Slack threads.",
	})
	adreChatErrorsTotal = prom.NewCounter(prom.CounterOpts{
		Name: "pmm_slack_adre_chat_errors_total",
		Help: "Failures of ADRE chat requests from the Slack integration.",
	})
	slackConnected = prom.NewGauge(prom.GaugeOpts{
		Name: "pmm_slack_adre_socket_connected",
		Help: "1 if Slack Socket Mode is connected on this leader.",
	})
	slackAuthzDeniedTotal = prom.NewCounterVec(prom.CounterOpts{
		Name: "pmm_slack_adre_authz_denied_total",
		Help: "Slack interactions denied by the human-chat allowlist, by path.",
	}, []string{"path"})
	slackRateLimitedTotal = prom.NewCounter(prom.CounterOpts{
		Name: "pmm_slack_adre_rate_limited_total",
		Help: "Slack human-chat turns rejected by the per-user rate limit.",
	})
)

func init() {
	prom.MustRegister(slackEventsTotal, adreChatSeconds, slackUploadsTotal, adreChatErrorsTotal, slackConnected,
		slackAuthzDeniedTotal, slackRateLimitedTotal)
}
