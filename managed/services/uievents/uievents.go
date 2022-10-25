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

// Package uievents provides facility to store UI events.
package uievents

import (
	"context"
	"encoding/json"
	"github.com/HdrHistogram/hdrhistogram-go"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/percona/pmm/managed/services/telemetry"
	"github.com/sirupsen/logrus"
	"sort"
	"time"

	uievents "github.com/percona/pmm/api/uieventspb"
)

const cleanupAfterHours = 30
const cleanupCheckInterval = 1 * time.Minute

// Service provides facility for storing UI events.
type Service struct {
	l              *logrus.Entry
	lastCleanup    time.Time
	dashboardUsage map[string]*DashboardUsageStat

	uievents.UnimplementedUIEventsServer
}

type DashboardUsageStat struct {
	title    string
	uid      string
	useCount int32
	loadTime *hdrhistogram.Histogram
}

// New returns platform Service.
func New() (*Service, error) {
	l := logrus.WithField("component", "platform")

	s := Service{
		l:              l,
		dashboardUsage: make(map[string]*DashboardUsageStat),
		lastCleanup:    time.Now(),
	}

	return &s, nil
}

// ScheduleCleanup schedules internal clean for internal UI event storage.
func (s *Service) ScheduleCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(cleanupCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanupIfNeeded()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Service) FetchMetrics(ctx context.Context, report *telemetry.Config) ([]*pmmv1.ServerMetric_Metric, error) {
	var metrics []*pmmv1.ServerMetric_Metric

	metrics = s.processDashboardStat(metrics)

	return metrics, nil
}

func (s *Service) processDashboardStat(metrics []*pmmv1.ServerMetric_Metric) []*pmmv1.ServerMetric_Metric {
	type DashboardStat struct {
		TopDashboards []string `json:"top_dashboards"`
	}

	if len(s.dashboardUsage) > 0 {
		dashboardStat := &DashboardStat{}

		// sort by usage
		sortedDashboardUsageKeys := make([]string, 0, len(s.dashboardUsage))
		for key := range s.dashboardUsage {
			sortedDashboardUsageKeys = append(sortedDashboardUsageKeys, key)
		}
		sort.SliceStable(sortedDashboardUsageKeys, func(i, j int) bool {
			return s.dashboardUsage[sortedDashboardUsageKeys[i]].useCount > s.dashboardUsage[sortedDashboardUsageKeys[j]].useCount
		})

		for i := 0; i < len(sortedDashboardUsageKeys); i++ {
			sortedDashboardUsageKey := sortedDashboardUsageKeys[i]
			stat := s.dashboardUsage[sortedDashboardUsageKey]
			dashboardStat.TopDashboards = append(dashboardStat.TopDashboards, stat.uid)
		}

		marshal, err := json.Marshal(dashboardStat)
		if err != nil {
			s.l.Error("failed to marshal to JSON")
		}
		metrics = append(metrics, &pmmv1.ServerMetric_Metric{
			Key:   "ui-events-dashboards",
			Value: string(marshal),
		})
	}
	return metrics
}

// Store stores metrics for further processing and sending to Portal.
func (s *Service) Store(ctx context.Context, request *uievents.StoreRequest) (*uievents.StoreResponse, error) {
	for _, dashboardUsageEvent := range request.DashboardUsage {
		stat, ok := s.dashboardUsage[dashboardUsageEvent.Uid]
		if !ok {
			stat = &DashboardUsageStat{
				title:    dashboardUsageEvent.Title,
				uid:      dashboardUsageEvent.Uid,
				useCount: 0,
				loadTime: s.loadTimeHistogram(),
			}
			s.dashboardUsage[dashboardUsageEvent.Uid] = stat
		}
		stat.useCount = stat.useCount + 1
		err := stat.loadTime.RecordValue(int64(dashboardUsageEvent.LoadTime))
		if err != nil {
			s.l.Error("failed to record value", err)
		}
	}
	return &uievents.StoreResponse{}, nil
}

func (s *Service) loadTimeHistogram() *hdrhistogram.Histogram {
	lowersValue := int64(0)
	highestValue := int64(1_000 * 60 * 10) // 10 min
	numberOfSignificantValueDigits := 5
	return hdrhistogram.New(lowersValue, highestValue, numberOfSignificantValueDigits)
}

func (s *Service) cleanupIfNeeded() {
	if time.Now().Sub(s.lastCleanup).Hours() > cleanupAfterHours {
		s.dashboardUsage = make(map[string]*DashboardUsageStat)
		s.lastCleanup = time.Now()
	}
}
