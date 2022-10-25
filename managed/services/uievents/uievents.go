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
	l               *logrus.Entry
	lastCleanup     time.Time
	dashboardUsage  map[string]*DashboardUsageStat
	componentsUsage map[string]*ComponentsUsageStat

	uievents.UnimplementedUIEventsServer
}

type DashboardUsageStat struct {
	title    string
	uid      string
	useCount int32
	loadTime *hdrhistogram.Histogram
}

type ComponentsUsageStat struct {
	uid      string
	useCount int32
	loadTime *hdrhistogram.Histogram
}

// New returns platform Service.
func New() (*Service, error) {
	l := logrus.WithField("component", "platform")

	s := Service{
		l:               l,
		dashboardUsage:  make(map[string]*DashboardUsageStat),
		componentsUsage: make(map[string]*ComponentsUsageStat),
		lastCleanup:     time.Now(),
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
	metrics = s.processComponentsStat(metrics)

	return metrics, nil
}

func (s *Service) processDashboardStat(metrics []*pmmv1.ServerMetric_Metric) []*pmmv1.ServerMetric_Metric {
	type Stat struct {
		TopDashboards         []string `json:"top_dashboards"`
		SlowDashboardsP95_1s  []string `json:"slow_dashboards_p95_1s"`
		SlowDashboardsP95_5s  []string `json:"slow_dashboards_p95_5s"`
		SlowDashboardsP95_10s []string `json:"slow_dashboards_p95_10s"`
	}

	if len(s.dashboardUsage) > 0 {
		dashboardStat := &Stat{
			SlowDashboardsP95_1s: []string{},
			SlowDashboardsP95_5s: []string{},
			SlowDashboardsP95_10s: []string{},
		}

		// TopDashboards
		keys := make([]string, 0, len(s.dashboardUsage))
		for key := range s.dashboardUsage {
			keys = append(keys, key)
		}
		sort.SliceStable(keys, func(i, j int) bool {
			return s.dashboardUsage[keys[i]].useCount > s.dashboardUsage[keys[j]].useCount
		})
		for i := 0; i < len(keys); i++ {
			sortedKey := keys[i]
			stat := s.dashboardUsage[sortedKey]
			dashboardStat.TopDashboards = append(dashboardStat.TopDashboards, stat.uid)
		}

		// SlowDashboardsP95
		sort.SliceStable(keys, func(i, j int) bool {
			return s.dashboardUsage[keys[i]].loadTime.ValueAtPercentile(95) > s.dashboardUsage[keys[j]].loadTime.ValueAtPercentile(95)
		})
		for i := 0; i < len(keys); i++ {
			sortedKey := keys[i]
			stat := s.dashboardUsage[sortedKey]
			p95 := stat.loadTime.ValueAtPercentile(95)
			if p95 >= 1_000 {
				dashboardStat.SlowDashboardsP95_1s = append(dashboardStat.SlowDashboardsP95_1s, stat.uid)
			}
			if p95 >= 5_000 {
				dashboardStat.SlowDashboardsP95_5s = append(dashboardStat.SlowDashboardsP95_5s, stat.uid)
			}
			if p95 >= 10_000 {
				dashboardStat.SlowDashboardsP95_10s = append(dashboardStat.SlowDashboardsP95_10s, stat.uid)
			}
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

func (s *Service) processComponentsStat(metrics []*pmmv1.ServerMetric_Metric) []*pmmv1.ServerMetric_Metric {
	type Stat struct {
		SlowComponentsP95_1s  []string `json:"slow_components_p95_1s"`
		SlowComponentsP95_5s  []string `json:"slow_components_p95_5s"`
		SlowComponentsP95_10s []string `json:"slow_components_p95_10s"`
	}

	if len(s.componentsUsage) > 0 {
		componentsStat := &Stat{
			SlowComponentsP95_1s:  []string{},
			SlowComponentsP95_5s:  []string{},
			SlowComponentsP95_10s: []string{},
		}

		// SlowComponentP95
		keys := make([]string, 0, len(s.componentsUsage))
		for key := range s.componentsUsage {
			keys = append(keys, key)
		}
		sort.SliceStable(keys, func(i, j int) bool {
			return s.componentsUsage[keys[i]].loadTime.ValueAtPercentile(95) > s.componentsUsage[keys[j]].loadTime.ValueAtPercentile(95)
		})
		for i := 0; i < len(keys); i++ {
			sortedKey := keys[i]
			stat := s.componentsUsage[sortedKey]
			p95 := stat.loadTime.ValueAtPercentile(95)
			if p95 >= 1_000 {
				componentsStat.SlowComponentsP95_1s = append(componentsStat.SlowComponentsP95_1s, stat.uid)
			}
			if p95 >= 5_000 {
				componentsStat.SlowComponentsP95_5s = append(componentsStat.SlowComponentsP95_5s, stat.uid)
			}
			if p95 >= 10_000 {
				componentsStat.SlowComponentsP95_10s = append(componentsStat.SlowComponentsP95_10s, stat.uid)
			}
		}

		marshal, err := json.Marshal(componentsStat)
		if err != nil {
			s.l.Error("failed to marshal to JSON")
		}
		metrics = append(metrics, &pmmv1.ServerMetric_Metric{
			Key:   "ui-events-components",
			Value: string(marshal),
		})
	}
	return metrics
}

// Store stores metrics for further processing and sending to Portal.
func (s *Service) Store(ctx context.Context, request *uievents.StoreRequest) (*uievents.StoreResponse, error) {
	for _, event := range request.DashboardUsage {
		stat, ok := s.dashboardUsage[event.Uid]
		if !ok {
			stat = &DashboardUsageStat{
				title:    event.Title,
				uid:      event.Uid,
				useCount: 0,
				loadTime: s.loadTimeHistogram(),
			}
			s.dashboardUsage[event.Uid] = stat
		}
		stat.useCount = stat.useCount + 1
		err := stat.loadTime.RecordValue(int64(event.LoadTime))
		if err != nil {
			s.l.Error("failed to record value", err)
		}
	}
	for _, event := range request.Fetching {
		stat, ok := s.componentsUsage[event.Component]
		if !ok {
			stat = &ComponentsUsageStat{
				uid:      event.Component,
				useCount: 0,
				loadTime: s.loadTimeHistogram(),
			}
			s.componentsUsage[event.Component] = stat
		}
		stat.useCount = stat.useCount + 1
		err := stat.loadTime.RecordValue(int64(event.LoadTime))
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
		s.componentsUsage = make(map[string]*ComponentsUsageStat)
		s.lastCleanup = time.Now()
	}
}
