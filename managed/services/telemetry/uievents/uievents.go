// Copyright (C) 2023 Percona LLC
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
	telemetryv1 "github.com/percona/saas/gen/telemetry/generic"
	"github.com/sirupsen/logrus"

	uieventsv1 "github.com/percona/pmm/api/uievents/v1"
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
	userFlowEvents  []*uieventsv1.UserFlowEvent

	stateM sync.RWMutex

	uieventsv1.UnimplementedUIEventsServiceServer
}

// DashboardUsageStat represents a structure for dashboard usage statistics.
type DashboardUsageStat struct {
	title    string
	uid      string
	useCount int32
	loadTime *hdrhistogram.Histogram
}

// ComponentsUsageStat represents a structure for component usage statistics.
type ComponentsUsageStat struct {
	uid      string
	useCount int32
	loadTime *hdrhistogram.Histogram
}

// New returns platform Service.
func New() *Service {
	l := logrus.WithField("component", "uievents")

	s := Service{
		l:               l,
		dashboardUsage:  make(map[string]*DashboardUsageStat),
		componentsUsage: make(map[string]*ComponentsUsageStat),
	}

	return &s
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

// FetchMetrics fetches metrics for the service based on the provided context and telemetry configuration.
func (s *Service) FetchMetrics(_ context.Context, _ telemetry.Config) ([]*telemetryv1.GenericReport_Metric, error) { //nolint:unparam
	s.stateM.RLock()
	defer s.stateM.RUnlock()

	var result []*telemetryv1.GenericReport_Metric
	if metric := s.processDashboardMetrics(); metric != nil {
		result = append(result, metric)
	}
	if metric := s.processComponentMetrics(); metric != nil {
		result = append(result, metric)
	}
	if metrics := s.processUserFlowEvents(); metrics != nil {
		result = append(result, metrics...)
	}

	return result, nil
}

func (s *Service) processDashboardMetrics() *telemetryv1.GenericReport_Metric {
	type Stat struct {
		TopDashboards         []string `json:"top_dashboards"`
		SlowDashboardsP95_1s  []string `json:"slow_dashboards_p95_1s"`
		SlowDashboardsP95_5s  []string `json:"slow_dashboards_p95_5s"`
		SlowDashboardsP95_10s []string `json:"slow_dashboards_p95_10s"`
	}

	if len(s.dashboardUsage) == 0 {
		return nil
	}

	dashboardStat := &Stat{
		SlowDashboardsP95_1s:  []string{},
		SlowDashboardsP95_5s:  []string{},
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

	serializedDashboardStat, err := json.Marshal(dashboardStat)
	if err != nil {
		s.l.Error("failed to marshal to JSON")
	}

	return &telemetryv1.GenericReport_Metric{
		Key:   "ui-events-dashboards",
		Value: string(serializedDashboardStat),
	}
}

func (s *Service) processComponentMetrics() *telemetryv1.GenericReport_Metric {
	type Stat struct {
		SlowComponentsP95_1s  []string `json:"slow_components_p95_1s"`
		SlowComponentsP95_5s  []string `json:"slow_components_p95_5s"`
		SlowComponentsP95_10s []string `json:"slow_components_p95_10s"`
	}

	if len(s.componentsUsage) != 0 {
		return nil
	}

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

	serializedComponentsStat, err := json.Marshal(componentsStat)
	if err != nil {
		s.l.Error("failed to marshal to JSON")
	}
	return &telemetryv1.GenericReport_Metric{
		Key:   "ui-events-components",
		Value: string(serializedComponentsStat),
	}
}

func (s *Service) processUserFlowEvents() []*telemetryv1.GenericReport_Metric {
	result := make([]*telemetryv1.GenericReport_Metric, 0, len(s.userFlowEvents))
	for _, event := range s.userFlowEvents {
		marshal, err := json.Marshal(event)
		if err != nil {
			s.l.Error("failed to marshal to JSON")
		}
		result = append(result, &telemetryv1.GenericReport_Metric{
			Key:   "ui-events-user-flow",
			Value: string(marshal),
		})
	}
	return result
}

// Store stores metrics for further processing and sending to Portal.
func (s *Service) Store(_ context.Context, request *uieventsv1.StoreRequest) (*uieventsv1.StoreResponse, error) { //nolint:unparam
	s.stateM.Lock()
	defer s.stateM.Unlock()

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
		stat.useCount++
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
		stat.useCount++
		err := stat.loadTime.RecordValue(int64(event.LoadTime))
		if err != nil {
			s.l.Error("failed to record value", err)
		}
	}

	s.userFlowEvents = append(s.userFlowEvents, request.UserFlowEvents...)

	return &uieventsv1.StoreResponse{}, nil
}

func (s *Service) cleanup() {
	s.stateM.Lock()
	defer s.stateM.Unlock()

	s.dashboardUsage = make(map[string]*DashboardUsageStat)
	s.componentsUsage = make(map[string]*ComponentsUsageStat)
	s.userFlowEvents = []*uieventsv1.UserFlowEvent{}
}
