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

package adre

import prom "github.com/prometheus/client_golang/prometheus"

var (
	holmesUsageCostTotal = prom.NewCounterVec(prom.CounterOpts{ //nolint:promlinter
		Namespace: "pmm", //nolint:goconst
		Subsystem: "holmes",
		Name:      "usage_total_cost",
		Help:      "Accumulated Holmes usage cost in USD by feature.",
	}, []string{"feature"})
	holmesUsageTokensTotal = prom.NewCounterVec(prom.CounterOpts{
		Namespace: "pmm",
		Subsystem: "holmes",
		Name:      "usage_tokens_total",
		Help:      "Accumulated Holmes total tokens by feature.",
	}, []string{"feature"})
)

func init() {
	prom.MustRegister(holmesUsageCostTotal, holmesUsageTokensTotal)
}

func observeHolmesUsage(feature string, usage *HolmesUsage) {
	if usage == nil || feature == "" {
		return
	}
	if usage.TotalCost != nil {
		holmesUsageCostTotal.WithLabelValues(feature).Add(*usage.TotalCost)
	}
	if usage.TotalTokens != nil {
		holmesUsageTokensTotal.WithLabelValues(feature).Add(float64(*usage.TotalTokens))
	}
}
