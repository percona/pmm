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

// The common currency every collection source deals in.
//
// Collection reads the same MongoDB estate from several places -- PMM's inventory,
// VictoriaMetrics, and (later) a SEP app running probes on Nomad clients -- and each
// knows a different, overlapping subset of the truth. Rather than special-case the
// sources against each other, every one of them emits the same thing: flat Fact records
// keyed by (service, field). Adding a source is then implementing one interface and
// changes nothing downstream.
//
// Merging is by declared precedence per field, never by call order. That is what makes
// "who wins" a piece of configuration rather than an accident of which source happened
// to run last, and it is why every merged field keeps its provenance: a document that
// cannot say where a version came from cannot distinguish "the node reports 7.0.39" from
// "the node reported 7.0.39 nine days ago and has been unreachable since".
//
// The field names and the precedence table are kept in step with om_inventory, whose
// facts merge here.

// SourceStatus says how completely one source answered.
//
// Recorded per source on the run, so a thin document is legible rather than merely thin:
// a snapshot assembled with metrics=OK, probe=FAILED is still correct about every version
// and honest about reachability.
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

// Source keys. These are the strings the precedence table and the run receipt use, and
// they are part of the contract with anything that contributes facts.
const (
	sourceInventory = "inventory"
	sourceMetrics   = "metrics"
	// The SEP inventory app's on-host facts -- argv, config paths, the installed binary
	// version. Produced by probeSource in probe_source.go, which reads them off SEP's
	// om_inventory app rather than probing anything itself; see that file for why.
	sourceProbe = "probe"
)

// Fact is keyed by PMM service, deliberately, not by host.
//
// A host with three mongods and no service PMM has registered -- the case this product
// exists to describe -- cannot contribute a fact the merge will keep, and neither can a
// naturally per-host probe attribute like repo_reachable or executor_reachable. That is a
// real gap and a decided one, not an oversight: host attributes live only on
// /v1/om/inventory/hosts (inventory.go), reached through a separate API rather than
// folded into this document. The alternative -- widening Fact with an optional node key
// so the merge becomes (node|service, field) -- was considered and deferred, because
// projection would then have to decide whether a host fact attaches to every service on
// that node or to some new slot, which is bigger than a fact-source addition. Until that
// lands, a naturally-per-host probe attribute must not be stuffed into every service row
// on its node: the copies would disagree the moment one of them changed.
//
// Fact carries one field about one service, as one source saw it.
type Fact struct {
	// Service is the PMM service ID this is about.
	Service string
	// Field is the document field this sets, e.g. "version".
	Field string
	// Value is the observed value.
	Value any
	// Source is the source key that produced it.
	Source string
	// ObservedAt is when the underlying observation was taken, or nil for a source that
	// is not time-bounded. Inventory is not time-bounded: it is current by definition. A
	// metric sample very much is.
	ObservedAt *time.Time
}

// MergedField is one field after precedence has picked a winner, with its provenance.
type MergedField struct {
	Value      any
	Source     string
	ObservedAt *time.Time
}

// age returns how old the observation is at now, and whether it is datable at all.
func (m MergedField) age(now time.Time) (time.Duration, bool) {
	if m.ObservedAt == nil {
		return 0, false
	}
	return now.Sub(*m.ObservedAt), true
}

// SourceResult reports everything one source produced in one collection run.
type SourceResult struct {
	// Source is the source key.
	Source string
	// Status is how completely it answered.
	Status SourceStatus
	// Facts is every fact it produced.
	Facts []Fact
	// Detail holds source-specific counters and errors, and lands verbatim in the run
	// receipt. This is what makes a thin snapshot legible.
	Detail map[string]any
	// Errors are per-service or per-query failures worth surfacing on the run.
	Errors []RunError
}

// RunError describes one thing that went wrong, scoped to whatever it concerns.
type RunError struct {
	Scope       string
	ServiceName string
	Code        string
	Message     string
}

// precedenceDefaultKey is the key under which the table holds the fallback ordering used
// by any field that does not name one of its own.
const precedenceDefaultKey = "default"

// defaultPrecedence declares, per field, which sources may set it and in what order.
//
// A source not listed for a field is forbidden from setting it, which is the point: it
// makes "the probe must never override the exporter's idea of replica-set state" a line
// in a table rather than a convention someone has to remember.
//
//   - version -- metrics first: mongodb_version_info is the running server's own report.
//     The probe sees the same thing from the node and is the fallback.
//   - installed_version -- probe only. The binary on disk is invisible to any metric,
//     and it is the whole upgrade-readiness signal.
//   - vendor / edition -- mongodb_version_info is the only source of either anywhere.
//   - endpoint -- member_idx is how the replica set itself addresses the member, which
//     beats inventory's record of where PMM reached the agent. Inventory stays the
//     fallback, and is the only source for a mongos, which has no member_idx.
//   - state -- metrics first, because a member's state as reported through the exporter
//     covers services the probe cannot reach at all.
//   - the exporter-only concepts -- reachability, load, role -- name metrics alone.
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
	fieldCPUUsage:        {sourceMetrics},
	fieldConnectionsFree: {sourceMetrics},
	fieldClusterRole:     {sourceMetrics},
	fieldIsMongos:        {sourceMetrics},
}

// mergeFacts folds every source's facts into one merged view per service.
//
// For each (service, field) the winner is the fact from the earliest source listed for
// that field. Facts from a source the field does not list are dropped rather than
// silently accepted, and a nil value never wins: a source that looked and saw nothing
// must not shadow one that looked and saw something.
//
// Ties at the same rank are broken by recency, which matters more than it sounds. One
// service can carry several series of the same metric -- a replica-set reconfiguration
// leaves the superseded mongodb_members_self behind, and it survives in the lookback
// window alongside the live one. Measured here: one member reported PRIMARY at 0s,
// SECONDARY at 7500s and PRIMARY at 68477s, all under its current service_id. Without
// this rule the document takes whichever the query happened to return first, and calls a
// live primary a secondary.
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

// supersedes reports whether an incoming fact should displace the one already held.
//
// Rank decides first. At equal rank -- the same source reporting the same field twice,
// which is what several series for one service look like -- the newer observation wins.
// An undatable fact never displaces a datable one, and never displaces its own kind:
// with nothing to compare, first seen stays.
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

// fieldSet is one service's merged fields plus the freshness rule to read them under.
//
// The rule travels with the data rather than sitting in a package variable: it is
// derived per run from PMM's configured scrape resolution, and a projection that read it
// from somewhere else could disagree with the collection it is projecting.
type fieldSet struct {
	fields map[string]MergedField
	now    time.Time
	maxAge time.Duration
}

// str returns a string field, or "" when it is unset, stale, or of another type.
func (f fieldSet) str(field string) string {
	value, ok := f.live(field)
	if !ok {
		return ""
	}
	s, _ := value.(string)
	return s
}

// f64 returns a float field, or nil when it is unset or stale.
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

// truthy reports whether a boolean field is set, fresh and true.
func (f fieldSet) truthy(field string) bool {
	value, ok := f.live(field)
	if !ok {
		return false
	}
	v, _ := value.(bool)
	return v
}

// live returns a field's value, dropping it when it is a volatile field whose
// observation has aged out.
//
// This is the half SEP records but does not act on. Facts are collected over a long
// window and kept with their age -- that is what tells "this service is gone" apart from
// "this service has not been scraped since Tuesday" -- but a volatile fact past its age
// must not be read as current. A replica-set state or an up flag from an hour ago is not
// a fact about now; an edition is.
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

// newestObservedAt returns the most recent observation among this row's live fields, or
// nil when none of them carry one at all.
//
// This is the row's own provenance, as against Snapshot.observed_at which is the newest
// across the whole estate and so cannot tell "this row is current" from "this row merely
// did not drag the snapshot's headline age down". An inventory-only fact has no
// ObservedAt to begin with, and a volatile fact that aged out is excluded the same way
// live() excludes it from a read -- a fact too old to read as current should not read as
// current here either.
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

// staleVolatile counts the volatile fields dropped for age, which the run receipt reports
// so a thin document says why it is thin.
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
