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
	"sort"
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"

	uievents "github.com/percona/pmm/api/uieventspb"
	"github.com/percona/pmm/managed/services/telemetry"
)

const (
	cleanupInterval                = 30 * time.Hour
	lowersValue                    = int64(0)
	highestValue                   = int64(1_000 * 60 * 10) // 10 min
	numberOfSignificantValueDigits = 5
)

// Service provides facility for storing UI events.
type Service struct {
	l               *logrus.Entry
	dashboardUsage  map[string]*DashboardUsageStat
	componentsUsage map[string]*ComponentsUsageStat
	userFlowEvents  []*uievents.UserFlowEvent

	stateLock sync.Mutex

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
	}

	return &s, nil
}

// ScheduleCleanup schedules internal clean for internal UI event storage.
func (s *Service) ScheduleCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanup()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Service) FetchMetrics(ctx context.Context, config telemetry.Config) ([]*pmmv1.ServerMetric_Metric, error) {
	s.stateLock.Lock()
	dashboardUsage := s.dashboardUsage
	componentsUsage := s.componentsUsage
	userFlowEvents := s.userFlowEvents
	s.stateLock.Unlock()

	var result []*pmmv1.ServerMetric_Metric
	if metric := s.processDashboardMetrics(dashboardUsage); metric != nil {
		result = append(result, metric)
	}
	if metric := s.processComponentMetrics(componentsUsage); metric != nil {
		result = append(result, metric)
	}
	if metrics := s.processUserFlowEvents(userFlowEvents); metrics != nil {
		result = append(result, metrics...)
	}

	return result, nil
}

func (s *Service) processDashboardMetrics(dashboardUsage map[string]*DashboardUsageStat) *pmmv1.ServerMetric_Metric {
	type Stat struct {
		TopDashboards         []string `json:"top_dashboards"`
		SlowDashboardsP95_1s  []string `json:"slow_dashboards_p95_1s"`
		SlowDashboardsP95_5s  []string `json:"slow_dashboards_p95_5s"`
		SlowDashboardsP95_10s []string `json:"slow_dashboards_p95_10s"`
	}

	if len(dashboardUsage) == 0 {
		return nil
	}

	dashboardStat := &Stat{
		SlowDashboardsP95_1s:  []string{},
		SlowDashboardsP95_5s:  []string{},
		SlowDashboardsP95_10s: []string{},
	}

	// TopDashboards
	keys := make([]string, 0, len(dashboardUsage))
	for key := range dashboardUsage {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return dashboardUsage[keys[i]].useCount > dashboardUsage[keys[j]].useCount
	})
	for i := 0; i < len(keys); i++ {
		sortedKey := keys[i]
		stat := dashboardUsage[sortedKey]
		dashboardStat.TopDashboards = append(dashboardStat.TopDashboards, stat.uid)
	}

	// SlowDashboardsP95
	sort.SliceStable(keys, func(i, j int) bool {
		return dashboardUsage[keys[i]].loadTime.ValueAtPercentile(95) > dashboardUsage[keys[j]].loadTime.ValueAtPercentile(95)
	})
	for i := 0; i < len(keys); i++ {
		sortedKey := keys[i]
		stat := dashboardUsage[sortedKey]
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

	serializedDashboardStat, err := json.Marshal(dashboardStat)
	if err != nil {
		s.l.Error("failed to marshal to JSON")
	}

	return &pmmv1.ServerMetric_Metric{
		Key:   "ui-events-dashboards",
		Value: string(serializedDashboardStat),
	}
}

func (s *Service) processComponentMetrics(componentsUsage map[string]*ComponentsUsageStat) *pmmv1.ServerMetric_Metric {
	type Stat struct {
		SlowComponentsP95_1s  []string `json:"slow_components_p95_1s"`
		SlowComponentsP95_5s  []string `json:"slow_components_p95_5s"`
		SlowComponentsP95_10s []string `json:"slow_components_p95_10s"`
	}

	if len(s.componentsUsage) > 0 {
		return nil
	}

	componentsStat := &Stat{
		SlowComponentsP95_1s:  []string{},
		SlowComponentsP95_5s:  []string{},
		SlowComponentsP95_10s: []string{},
	}

	// SlowComponentP95
	keys := make([]string, 0, len(componentsUsage))
	for key := range componentsUsage {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return componentsUsage[keys[i]].loadTime.ValueAtPercentile(95) > componentsUsage[keys[j]].loadTime.ValueAtPercentile(95)
	})
	for i := 0; i < len(keys); i++ {
		sortedKey := keys[i]
		stat := componentsUsage[sortedKey]
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

	serializedComponentsStat, err := json.Marshal(componentsStat)
	if err != nil {
		s.l.Error("failed to marshal to JSON")
	}
	return &pmmv1.ServerMetric_Metric{
		Key:   "ui-events-components",
		Value: string(serializedComponentsStat),
	}
}

func (s *Service) processUserFlowEvents(events []*uievents.UserFlowEvent) []*pmmv1.ServerMetric_Metric {
	var result []*pmmv1.ServerMetric_Metric
	for _, event := range events {
		marshal, err := json.Marshal(event)
		if err != nil {
			s.l.Error("failed to marshal to JSON")
		}
		result = append(result, &pmmv1.ServerMetric_Metric{
			Key:   "ui-events-user-flow",
			Value: string(marshal),
		})
	}
	return result
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
				loadTime: hdrhistogram.New(lowersValue, highestValue, numberOfSignificantValueDigits),
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
				loadTime: hdrhistogram.New(lowersValue, highestValue, numberOfSignificantValueDigits),
			}
			s.componentsUsage[event.Component] = stat
		}
		stat.useCount = stat.useCount + 1
		err := stat.loadTime.RecordValue(int64(event.LoadTime))
		if err != nil {
			s.l.Error("failed to record value", err)
		}
	}

	s.userFlowEvents = append(s.userFlowEvents, request.UserFlowEvents...)

	return &uievents.StoreResponse{}, nil
}

func (s *Service) cleanup() {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	s.dashboardUsage = make(map[string]*DashboardUsageStat)
	s.componentsUsage = make(map[string]*ComponentsUsageStat)
	s.userFlowEvents = []*uievents.UserFlowEvent{}
}
