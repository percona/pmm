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

package mcp

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

const (
	// The not_found text for a service that is not in the caller's service list.
	notInServiceList = "not in your service list (services that sent metrics in the window searched)"

	// The number of candidate services one getReport asks for.
	resolveReportLimit = 100
	// The number of candidates an ambiguity message names.
	maxListedCandidates = 20

	groupByServiceName = "service_name"
)

// queryRefInput is the input form of queryRef, with string times, so that a bad time is invalid_input.
type queryRefInput struct {
	QueryID    string `json:"queryid,omitempty" jsonschema:"Query id."`
	ServiceID  string `json:"service_id,omitempty" jsonschema:"Service id the query ran on."`
	PeriodFrom string `json:"period_from,omitempty" jsonschema:"Start of the window the query was seen in."`
	PeriodTo   string `json:"period_to,omitempty" jsonschema:"End of the window the query was seen in."`
}

// resolvedQuery is a query reference resolved to one service in the caller's list.
type resolvedQuery struct {
	Service  serviceInfo
	Ref      queryRef
	Database string
	Tables   []string
	// Example is the stored statement as QAN holds it, empty when QAN has none for this service.
	Example string
	// DetailErr is the failure of the metrics or example call that Database, Tables and Example come from.
	DetailErr error
}

// refFromInput builds a query reference from query_ref or from the flat queryid, service_id and period fields.
func refFromInput(ref *queryRefInput, queryID, serviceID, periodFrom, periodTo string, now time.Time) (queryRef, error) {
	flat := queryID != "" || serviceID != "" || periodFrom != "" || periodTo != ""
	switch {
	case ref == nil && !flat:
		return queryRef{}, newToolError(codeInvalidInput,
			"pass query_ref, as pmm_top_queries or pmm_query_detail return it, or pass queryid")
	case ref != nil && flat:
		return queryRef{}, newToolError(codeInvalidInput,
			"pass query_ref alone, without queryid, service_id, period_from or period_to; to choose a service, set service_id inside query_ref")
	case ref != nil:
		if ref.QueryID == "" || ref.PeriodFrom == "" || ref.PeriodTo == "" {
			return queryRef{}, newToolError(codeInvalidInput,
				"query_ref needs queryid, period_from and period_to; pass it as pmm_top_queries or pmm_query_detail return it")
		}
		queryID, serviceID, periodFrom, periodTo = ref.QueryID, ref.ServiceID, ref.PeriodFrom, ref.PeriodTo
	case queryID == "":
		return queryRef{}, newToolError(codeInvalidInput, "service_id, period_from and period_to need a queryid")
	}

	from, to, err := parseWindow(periodFrom, periodTo, now)
	if err != nil {
		return queryRef{}, err
	}
	return queryRef{QueryID: queryID, ServiceID: serviceID, PeriodFrom: from, PeriodTo: to, fromQueryRef: ref != nil}, nil
}

// resolveQuery finds the one service in the caller's list that the query ran on, with its database, tables and stored example.
func (s *Service) resolveQuery(ctx context.Context, auth callerAuth, ref queryRef) (*resolvedQuery, error) {
	err := checkQANValue("queryid", ref.QueryID)
	if err != nil {
		return nil, err
	}
	err = checkQANValue("service_id", ref.ServiceID)
	if err != nil {
		return nil, err
	}

	services, ids, err := s.callerServices(ctx, auth, ref.ServiceID, window{From: ref.PeriodFrom, To: ref.PeriodTo})
	if err != nil {
		return nil, err
	}

	candidates, total, err := s.queryCandidates(ctx, auth, ref, services, ids)
	if err != nil {
		return nil, err
	}

	w := window{From: ref.PeriodFrom, To: ref.PeriodTo}
	widen := "pass period_from and period_to covering when it ran, or its query_ref from pmm_top_queries"
	if ref.fromQueryRef {
		widen = "pass a query_ref whose period covers when it ran"
	}
	switch {
	case len(candidates) == 0 && ref.ServiceID != "":
		return nil, newToolError(codeNotFound, "the query was not seen on that service in %s; %s", windowText(w), widen)
	case len(candidates) == 0:
		return nil, newToolError(codeNotFound, "the query was not seen on any of your services in %s; %s", windowText(w), widen)
	case len(candidates) > 1:
		return nil, newToolError(codeInvalidInput, "%s", ambiguousMessage(candidates, total, ref.fromQueryRef))
	}

	svc := candidates[0]
	ref.ServiceID = svc.ServiceID
	out := &resolvedQuery{Service: svc, Ref: ref}

	d, err := s.fetchQueryDetail(ctx, auth, queryDetailInput{QueryID: ref.QueryID, ServiceID: svc.ServiceID}, []string{svc.ServiceID}, ref.PeriodFrom, ref.PeriodTo)
	if err != nil {
		return nil, err
	}
	out.DetailErr = cmp.Or(d.metricsErr, d.exampleErr)

	var schema, database string
	if d.metrics != nil && d.metrics.Metadata != nil {
		m := d.metrics.Metadata
		if m.ServiceName == "" || m.ServiceName == svc.ServiceName {
			schema, database = m.Schema, m.Database
		}
	}
	example := d.example
	if example != nil && example.ServiceID != svc.ServiceID {
		example = nil
	}
	if example != nil {
		out.Example = example.Example
		if svc.Engine == enginePostgreSQL {
			out.Tables = example.Tables
		}
	}
	switch svc.Engine {
	case engineMySQL:
		out.Database = schema
		if out.Database == "" && example != nil {
			out.Database = example.Schema
		}
	default:
		out.Database = database
	}
	return out, nil
}

// errNotInList is the not_found for a service outside the service list of the window, naming the window and no service.
func errNotInList(w window, hint string) *toolError {
	span, cut := listSpan(w)
	text := notInServiceList + "; searched " + windowText(span)
	if cut {
		text += ", the last 7 days of the call's window"
	}
	return newToolError(codeNotFound, "%s%s", text, hint)
}

// callerServices returns the caller's services and the ids of those that QAN may be asked about: the given one,
// which must be in the list, else every one that passes checkQANValue.
func (s *Service) callerServices(ctx context.Context, auth callerAuth, serviceID string, w window) ([]serviceInfo, []string, error) {
	services, err := s.listServices(ctx, auth, w)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(services))
	for _, svc := range services {
		if checkQANValue("service_id", svc.ServiceID) == nil {
			ids = append(ids, svc.ServiceID)
		}
	}
	if serviceID != "" {
		if !slices.Contains(ids, serviceID) {
			return nil, nil, errNotInList(w, "")
		}
		ids = []string{serviceID}
	}
	if len(ids) == 0 {
		return nil, nil, errNotInList(w, "")
	}
	return services, ids, nil
}

// queryCandidates returns the services of the caller's list that the query was seen on, and the report's total row count.
func (s *Service) queryCandidates(ctx context.Context, auth callerAuth, ref queryRef, services []serviceInfo, ids []string) ([]serviceInfo, int64, error) {
	report, err := s.api.GetReport(ctx, auth, qan_service.GetReportBody{
		PeriodStartFrom: strfmt.DateTime(ref.PeriodFrom),
		PeriodStartTo:   strfmt.DateTime(ref.PeriodTo),
		GroupBy:         groupByServiceName,
		OrderBy:         "-load",
		Limit:           resolveReportLimit,
		Columns:         slices.Clone(baseColumns),
		MainMetric:      "load",
		Labels: []*qan_service.GetReportParamsBodyLabelsItems0{
			{Key: groupByQuery, Value: []string{ref.QueryID}},
			{Key: "service_id", Value: ids},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	asked := slices.DeleteFunc(slices.Clone(services), func(svc serviceInfo) bool { return !slices.Contains(ids, svc.ServiceID) })
	var out []serviceInfo
	for _, row := range report.Rows {
		if row.Dimension == "" {
			continue
		}
		svc, ok := serviceNamed(asked, row.Dimension)
		if !ok {
			continue
		}
		if slices.ContainsFunc(out, func(o serviceInfo) bool { return o.ServiceID == svc.ServiceID }) {
			continue
		}
		out = append(out, svc)
	}
	return out, report.TotalRows, nil
}

// ambiguousMessage lists the candidate services and asks for a service_id where the caller's input form takes one.
func ambiguousMessage(candidates []serviceInfo, total int64, fromQueryRef bool) string {
	listed := candidates[:min(len(candidates), maxListedCandidates)]
	names := make([]string, 0, len(listed))
	for _, svc := range listed {
		names = append(names, fmt.Sprintf("%s (%s)", svc.ServiceName, svc.ServiceID))
	}
	text := "the query was seen on several services: " + strings.Join(names, ", ")
	if more := total - int64(len(listed)); len(candidates) > len(listed) && more > 0 {
		text += fmt.Sprintf(", and %d more", more)
	}
	if fromQueryRef {
		return text + "; set service_id inside query_ref to one of these"
	}
	return text + "; pass service_id to choose one"
}

// windowText renders a window for a message.
func windowText(w window) string {
	return w.From.UTC().Format(time.RFC3339) + " – " + w.To.UTC().Format(time.RFC3339)
}
