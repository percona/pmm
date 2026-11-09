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

package om

import (
	"slices"
	"time"
)

// SourceStatus says how completely one source answered.
type SourceStatus string

// The states a source can report.
const (
	// SourceOK means the source answered for every service asked about.
	SourceOK SourceStatus = "ok"
	// SourcePartial means it answered for some services but not all.
	SourcePartial SourceStatus = "partial"
	// SourceFailed means it answered for none, or errored.
	SourceFailed SourceStatus = "failed"
	// SourceDisabled means it was switched off and never ran.
	SourceDisabled SourceStatus = "disabled"
)

// Source keys, part of the contract with anything that contributes facts.
const (
	sourceInventory = "inventory"
	sourceMetrics   = "metrics"
	// On-host facts (argv, config path, installed version), read from the PMM Extensions om_inventory app.
	sourceProbe = "probe"
)

// Fact carries one field about one service, as one source saw it.
// Facts are keyed by PMM service, not host; per-host attributes belong on /v1/om/inventory/hosts.
type Fact struct {
	// Service is the PMM service ID.
	Service string
	Field   string
	Value   any
	Source  string
	// ObservedAt is when the observation was taken, or nil for a source that is not time-bounded, such as inventory.
	ObservedAt *time.Time
}

// MergedField is one field after precedence has picked a winner, with its provenance.
type MergedField struct {
	Value      any
	Source     string
	ObservedAt *time.Time
}

func (m MergedField) age(now time.Time) (time.Duration, bool) {
	if m.ObservedAt == nil {
		return 0, false
	}
	return now.Sub(*m.ObservedAt), true
}

// SourceResult reports everything one source produced in one collection run.
type SourceResult struct {
	Source string
	Status SourceStatus
	Facts  []Fact
	// Detail holds source-specific counters and errors, and lands verbatim in the run receipt.
	Detail map[string]any
	Errors []RunError
}

// RunError describes one thing that went wrong, scoped to whatever it concerns.
type RunError struct {
	Scope       string
	ServiceName string
	Code        string
	Message     string
}

const precedenceDefaultKey = "default"

// defaultPrecedence lists, per field, the sources allowed to set it, highest precedence first.
// A source not listed for a field can never set it.
var defaultPrecedence = map[string][]string{
	precedenceDefaultKey: {sourceMetrics, sourceInventory, sourceProbe},

	fieldVersion:          {sourceMetrics, sourceProbe},
	fieldInstalledVersion: {sourceProbe},
	fieldConfigPath:       {sourceProbe},
	fieldArgv:             {sourceProbe},
	fieldVendor:           {sourceMetrics},
	fieldEdition:          {sourceMetrics},
	fieldEndpoint:         {sourceMetrics, sourceInventory},
	fieldState:            {sourceMetrics, sourceProbe},

	fieldReplicationLag: {sourceMetrics},
	fieldOplogHead:      {sourceMetrics},
	fieldOplogTail:      {sourceMetrics},

	fieldExporterUp:      {sourceMetrics},
	fieldLastUp:          {sourceMetrics},
	fieldCPUUsage:        {sourceMetrics},
	fieldConnectionsFree: {sourceMetrics},
	fieldClusterRole:     {sourceMetrics},
	fieldIsMongos:        {sourceMetrics},
}

// mergeFacts folds every source's facts into one view per service, by per-field precedence.
// Facts from unlisted sources are dropped, and a nil value never shadows a real one.
func mergeFacts(results []SourceResult, precedence map[string][]string) map[string]map[string]MergedField {
	merged := make(map[string]map[string]MergedField)

	for _, result := range results {
		for _, fact := range result.Facts {
			if fact.Value == nil || fact.Service == "" {
				continue
			}
			order, ok := precedence[fact.Field]
			if !ok {
				order = precedence[precedenceDefaultKey]
			}
			rank := slices.Index(order, fact.Source)
			if rank < 0 {
				continue
			}

			fields, ok := merged[fact.Service]
			if !ok {
				fields = make(map[string]MergedField)
				merged[fact.Service] = fields
			}
			if held, ok := fields[fact.Field]; ok && !supersedes(fact, held, order, rank) {
				continue
			}
			fields[fact.Field] = MergedField{
				Value:      fact.Value,
				Source:     fact.Source,
				ObservedAt: fact.ObservedAt,
			}
		}
	}
	return merged
}

// supersedes breaks equal-rank ties by recency: a replica-set reconfiguration leaves superseded
// series in the lookback window, and the first one returned may be stale.
func supersedes(fact Fact, held MergedField, order []string, rank int) bool {
	heldRank := slices.Index(order, held.Source)
	if heldRank != rank {
		return heldRank > rank
	}
	if fact.ObservedAt == nil {
		return false
	}
	if held.ObservedAt == nil {
		return true
	}
	return fact.ObservedAt.After(*held.ObservedAt)
}

// fieldSet carries its freshness rule because maxAge is derived per run from PMM's scrape resolution.
type fieldSet struct {
	fields map[string]MergedField
	now    time.Time
	maxAge time.Duration
}

func (f fieldSet) str(field string) string {
	value, ok := f.live(field)
	if !ok {
		return ""
	}
	s, _ := value.(string)
	return s
}

func (f fieldSet) f64(field string) *float64 {
	value, ok := f.live(field)
	if !ok {
		return nil
	}
	v, ok := value.(float64)
	if !ok {
		return nil
	}
	return &v
}

func (f fieldSet) truthy(field string) bool {
	value, ok := f.live(field)
	if !ok {
		return false
	}
	v, _ := value.(bool)
	return v
}

// live returns a field's value, dropping a volatile field whose observation has aged out.
func (f fieldSet) live(field string) (any, bool) {
	held, ok := f.fields[field]
	if !ok {
		return nil, false
	}
	if !volatileFields[field] {
		return held.Value, true
	}
	age, datable := held.age(f.now)
	if datable && age > f.maxAge {
		return nil, false
	}
	return held.Value, true
}

func (f fieldSet) newestObservedAt() *time.Time {
	var newest *time.Time
	for field, held := range f.fields {
		if held.ObservedAt == nil {
			continue
		}
		if volatileFields[field] {
			if _, ok := f.live(field); !ok {
				continue
			}
		}
		if newest == nil || held.ObservedAt.After(*newest) {
			newest = held.ObservedAt
		}
	}
	return newest
}

func (f fieldSet) staleVolatile() int {
	stale := 0
	for field, held := range f.fields {
		if !volatileFields[field] {
			continue
		}
		if age, datable := held.age(f.now); datable && age > f.maxAge {
			stale++
		}
	}
	return stale
}
