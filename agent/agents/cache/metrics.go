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

package cache

import "github.com/prometheus/client_golang/prometheus"

// MetricsFromStats builds Prometheus metrics from cache.Stats.
func MetricsFromStats(stats Stats, agentID string, cacheType string) []prometheus.Metric {
	metrics := []prometheus.Metric{
		prometheus.MustNewConstMetric(mCurrentDesc, prometheus.GaugeValue, float64(stats.Current), agentID, cacheType),
		prometheus.MustNewConstMetric(mUpdatedNDesc, prometheus.CounterValue, float64(stats.UpdatedN), agentID, cacheType),
		prometheus.MustNewConstMetric(mAddedNDesc, prometheus.CounterValue, float64(stats.AddedN), agentID, cacheType),
		prometheus.MustNewConstMetric(mRemovedNDesc, prometheus.CounterValue, float64(stats.RemovedN), agentID, cacheType),
		prometheus.MustNewConstMetric(mTrimmedNDesc, prometheus.CounterValue, float64(stats.TrimmedN), agentID, cacheType),
		prometheus.MustNewConstMetric(mOldestDesc, prometheus.CounterValue, float64(stats.Oldest.Unix()), agentID, cacheType),
		prometheus.MustNewConstMetric(mNewestDesc, prometheus.CounterValue, float64(stats.Newest.Unix()), agentID, cacheType),
	}
	return metrics
}

const (
	prometheusNamespace = "pmm_agent"
	prometheusSubsystem = "statements_cache"
)

var (
	mCurrentDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "current"),
		"Current number of rows in cache.",
		[]string{"agent_id", "cache_type"},
		nil)
	mUpdatedNDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "updated_n"),
		"Number of updated rows in cache.",
		[]string{"agent_id", "cache_type"},
		nil)
	mAddedNDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "added_n"),
		"Number of rows added to cache.",
		[]string{"agent_id", "cache_type"},
		nil)
	mRemovedNDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "removed_n"),
		"Number of rows removed from cache.",
		[]string{"agent_id", "cache_type"},
		nil)
	mTrimmedNDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "trimmed_n"),
		"Number of rows trimmed from cache.",
		[]string{"agent_id", "cache_type"},
		nil)
	mOldestDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "oldest"),
		"Timestamp of oldest row in cache.",
		[]string{"agent_id", "cache_type"},
		nil)
	mNewestDesc = prometheus.NewDesc(
		prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "newest"),
		"Timestamp of newest row in cache.",
		[]string{"agent_id", "cache_type"},
		nil)
)
