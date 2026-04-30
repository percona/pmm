// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

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
		Buckets: prom.ExponentialBuckets(0.5, 2, 14),
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
)

func init() {
	prom.MustRegister(slackEventsTotal, adreChatSeconds, slackUploadsTotal, adreChatErrorsTotal, slackConnected)
}
