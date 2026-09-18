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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

const (
	// DefaultActionTimeout bounds the EXPLAIN / SHOW CREATE TABLE polling.
	DefaultActionTimeout = 15 * time.Second

	pollInitialDelay = 300 * time.Millisecond
	pollMaxDelay     = 2 * time.Second
	pollBackoff      = 1.5

	formatJSON        = "json"
	formatTraditional = "traditional"
)

type explainInput struct {
	ServiceID    string   `json:"service_id" jsonschema:"Service id from pmm_inventory"`
	QueryID      string   `json:"queryid,omitempty" jsonschema:"Query id from pmm_top_queries (MySQL, PostgreSQL stored plans, MongoDB via the stored example)"`
	Query        string   `json:"query,omitempty" jsonschema:"Explicit statement to explain (MongoDB only; MySQL is addressed by queryid)"`
	Database     string   `json:"database,omitempty" jsonschema:"Schema the query runs in (from pmm_query_detail); MySQL only"`
	Placeholders []string `json:"placeholders,omitempty" jsonschema:"Values for ? placeholders in the fingerprint, in order; MySQL only"`
	Format       string   `json:"format,omitempty" jsonschema:"json (default) or traditional; MySQL only"`
}

type schemaInput struct {
	ServiceID      string `json:"service_id" jsonschema:"Service id from pmm_inventory"`
	Database       string `json:"database" jsonschema:"Database / schema name"`
	Table          string `json:"table,omitempty" jsonschema:"Table name"`
	TableName      string `json:"table_name,omitempty" jsonschema:"Alias of table"`
	IncludeIndexes bool   `json:"include_indexes,omitempty" jsonschema:"Also return SHOW INDEX output"`
}

func (s *Service) registerActionTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_get_explain",
		Title: "Execution plan",
		Description: "Return the execution plan for a query addressed by queryid. PostgreSQL plans come from " +
			"pg_stat_monitor's stored plan (needs pgsm_enable_query_plan=on); MySQL and MongoDB plans are " +
			"produced live by the PMM agent with a non-executing EXPLAIN (never EXPLAIN ANALYZE). If a MySQL " +
			"fingerprint has ? placeholders and no query example is stored, pass placeholders and database.",
		Annotations: readOnly("Execution plan"),
	}, handle(s, "pmm_get_explain", s.explain))

	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_get_schema",
		Title: "Table DDL",
		Description: "Return SHOW CREATE TABLE (MySQL or PostgreSQL) for a table via the PMM agent, " +
			"optionally with its indexes.",
		Annotations: readOnly("Table DDL"),
	}, handle(s, "pmm_get_schema", s.schema))
}

func (s *Service) explain(ctx context.Context, req *mcp.CallToolRequest, in explainInput) (*mcp.CallToolResult, error) {
	if in.ServiceID == "" {
		return nil, newToolError(codeInvalidInput, "service_id is required")
	}
	if in.QueryID == "" && in.Query == "" {
		return nil, newToolError(codeInvalidInput, "queryid or query is required")
	}
	format := in.Format
	if format == "" {
		format = formatJSON
	}
	if format != formatJSON && format != formatTraditional {
		return nil, newToolError(codeInvalidInput, "format must be json or traditional; got '%s'", in.Format)
	}
	auth := callerAuthFromHeader(req.Extra.Header)
	base := s.publicBaseURL(ctx, req.Extra.Header)
	now := s.now()

	// Probe the stored plan first: pg_stat_monitor stores one per digest when
	// pgsm_enable_query_plan is on. Live-confirmed on 3.8.1: 200 {} when absent.
	if in.QueryID != "" {
		plan, err := s.api.GetQueryPlan(ctx, auth, in.QueryID)
		if err != nil {
			s.l.WithField("tool", "pmm_get_explain").Debugf("Stored-plan probe failed: %s.", err)
		} else if plan != nil && plan.QueryPlan != "" {
			text := "Stored plan (pg_stat_monitor, planid " + plan.Planid + "):\n```\n" + plan.QueryPlan + "\n```"
			text += "\n\n" + link("View this query in PMM", qanQueryURL(base, in.QueryID, "", now.Add(-time.Hour), now, "explain"))
			return textResult(text), nil
		}
	}

	svc, err := s.engineOf(ctx, auth, in.ServiceID)
	if err != nil {
		return nil, err
	}

	var body actions_service.StartServiceActionBody
	switch svc.Engine {
	case engineMySQL:
		if in.QueryID == "" {
			return nil, newToolError(codeInvalidInput, "MySQL EXPLAIN is addressed by queryid; pass the queryid from pmm_top_queries")
		}
		// PMM resolves the placeholder values from the stored query example, so
		// service_id + queryid (+ database) is enough; explicit placeholders
		// override. Without an example PMM EXPLAINs the raw ? fingerprint and
		// MySQL answers 1064, mapped below.
		if format == formatTraditional {
			body.MysqlExplain = &actions_service.StartServiceActionParamsBodyMysqlExplain{
				ServiceID: in.ServiceID, QueryID: in.QueryID, Database: in.Database, Placeholders: in.Placeholders,
			}
		} else {
			body.MysqlExplainJSON = &actions_service.StartServiceActionParamsBodyMysqlExplainJSON{
				ServiceID: in.ServiceID, QueryID: in.QueryID, Database: in.Database, Placeholders: in.Placeholders,
			}
		}
	case engineMongoDB:
		query := in.Query
		if query == "" {
			query, err = s.storedExample(ctx, auth, in.QueryID, in.ServiceID, now)
			if err != nil {
				return nil, err
			}
		}
		body.MongodbExplain = &actions_service.StartServiceActionParamsBodyMongodbExplain{ServiceID: in.ServiceID, Query: query}
	case enginePostgreSQL:
		return nil, newToolError(codeNotFound,
			"no stored plan for queryid '%s' and PMM has no live EXPLAIN action for PostgreSQL. Monitor the service with "+
				"pg_stat_monitor and pg_stat_monitor.pgsm_enable_query_plan=on so plans are captured, then retry", in.QueryID)
	default:
		return nil, newToolError(codeInvalidInput, "EXPLAIN is not available for engine '%s'", svc.Engine)
	}

	output, err := s.runAction(ctx, auth, body)
	if err != nil {
		return nil, err
	}

	text := fmt.Sprintf("EXPLAIN (%s, %s %s):\n```\n%s\n```", format, svc.Engine, svc.ServiceName, decodeExplainOutput(output, s.rawSQL()))
	if in.QueryID != "" {
		text += "\n\n" + link("View this query in PMM", qanQueryURL(base, in.QueryID, svc.ServiceName, now.Add(-time.Hour), now, "explain"))
	}
	return textResult(text), nil
}

// storedExample returns the stored example statement for a queryid, which is
// the natural input of mongodb_explain.
func (s *Service) storedExample(ctx context.Context, auth callerAuth, queryID, serviceID string, now time.Time) (string, error) {
	if queryID == "" {
		return "", newToolError(codeInvalidInput, "queryid or query is required")
	}
	res, err := s.api.GetQueryExample(ctx, auth, qan_service.GetQueryExampleBody{
		PeriodStartFrom: strfmt.DateTime(now.Add(-day)),
		PeriodStartTo:   strfmt.DateTime(now),
		FilterBy:        queryID,
		GroupBy:         groupByQuery,
		Labels: []*qan_service.GetQueryExampleParamsBodyLabelsItems0{
			{Key: groupByQuery, Value: []string{queryID}},
			{Key: "service_id", Value: []string{serviceID}},
		},
		Limit: 1,
	})
	if err != nil {
		return "", err
	}
	if len(res.QueryExamples) == 0 || res.QueryExamples[0].Example == "" {
		return "", newToolError(codeNoQuerySource,
			"no stored query example for queryid '%s' in the last 24h; enable query examples for the service or pass an explicit query", queryID)
	}
	return res.QueryExamples[0].Example, nil
}

func (s *Service) schema(ctx context.Context, req *mcp.CallToolRequest, in schemaInput) (*mcp.CallToolResult, error) {
	table := in.Table
	if table == "" {
		table = in.TableName
	}
	if in.ServiceID == "" || in.Database == "" || table == "" {
		return nil, newToolError(codeInvalidInput, "service_id, database and table are required")
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	svc, err := s.engineOf(ctx, auth, in.ServiceID)
	if err != nil {
		return nil, err
	}

	var ddlBody, indexBody actions_service.StartServiceActionBody
	switch svc.Engine {
	case engineMySQL:
		ddlBody.MysqlShowCreateTable = &actions_service.StartServiceActionParamsBodyMysqlShowCreateTable{
			ServiceID: in.ServiceID, Database: in.Database, TableName: table,
		}
		indexBody.MysqlShowIndex = &actions_service.StartServiceActionParamsBodyMysqlShowIndex{
			ServiceID: in.ServiceID, Database: in.Database, TableName: table,
		}
	case enginePostgreSQL:
		ddlBody.PostgresShowCreateTable = &actions_service.StartServiceActionParamsBodyPostgresShowCreateTable{
			ServiceID: in.ServiceID, Database: in.Database, TableName: table,
		}
		indexBody.PostgresShowIndex = &actions_service.StartServiceActionParamsBodyPostgresShowIndex{
			ServiceID: in.ServiceID, Database: in.Database, TableName: table,
		}
	default:
		return nil, newToolError(codeInvalidInput, "SHOW CREATE TABLE is not available for engine '%s'", svc.Engine)
	}

	ddl, err := s.runAction(ctx, auth, ddlBody)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(ddl) == "" {
		return textResult("No DDL returned for that table."), nil
	}
	text := "```sql\n" + strings.TrimRight(ddl, "\n") + "\n```"

	if in.IncludeIndexes {
		indexes, err := s.runAction(ctx, auth, indexBody)
		if err != nil {
			return nil, err
		}
		text += "\n\nindexes:\n```\n" + strings.TrimRight(indexes, "\n") + "\n```"
	}
	return textResult(text), nil
}

// runAction starts a service action and polls it with 300 ms -> 2 s backoff
// until it is done or the action timeout elapses.
func (s *Service) runAction(ctx context.Context, auth callerAuth, body actions_service.StartServiceActionBody) (string, error) {
	started, err := s.api.StartServiceAction(ctx, auth, body)
	if err != nil {
		return "", mapActionStartError(err)
	}
	actionID := actionIDOf(started)
	if actionID == "" {
		return "", newToolError(codePMMUnavailable, "no action_id returned by startServiceAction")
	}

	timeout := s.actionTimeout()
	deadline := time.Now().Add(timeout)
	delay := pollInitialDelay
	for {
		res, err := s.api.GetAction(ctx, auth, actionID)
		if err != nil {
			return "", err
		}
		if res.Done {
			if res.Error != "" {
				return "", mapActionError(res.Error)
			}
			return res.Output, nil
		}
		if time.Now().After(deadline) {
			return "", newToolError(codeTimeout, "action %s did not complete within %s (PMM_MCP_ACTION_TIMEOUT)", actionID, timeout)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
		}
		delay = min(time.Duration(float64(delay)*pollBackoff), pollMaxDelay)
	}
}

// actionIDOf finds the action id under whichever oneof key PMM answered with.
func actionIDOf(res *actions_service.StartServiceActionOKBody) string {
	if res == nil {
		return ""
	}
	b, err := json.Marshal(res)
	if err != nil {
		return ""
	}
	var m map[string]struct {
		ActionID string `json:"action_id"`
	}
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	for _, v := range m {
		if v.ActionID != "" {
			return v.ActionID
		}
	}
	return ""
}

// mapActionStartError refines the generic HTTP mapping for startServiceAction:
// pmm-managed answers FailedPrecondition / NotFound when no pmm-agent can run
// the action, which is an agent problem rather than bad input.
func mapActionStartError(err error) error {
	te := mapError(err)
	msg := strings.ToLower(te.message)
	if strings.Contains(msg, "agent") && (te.code == codeInvalidInput || te.code == codeNotFound) {
		return newToolError(codeAgentUnreachable, "%s", te.message)
	}
	return te
}

// mapActionError classifies the error text of a finished action.
func mapActionError(actionErr string) *toolError {
	e := strings.ToLower(actionErr)
	switch {
	case strings.Contains(e, "denied") || strings.Contains(e, "privilege") || strings.Contains(e, "permission"):
		return newToolError(codeInsufficientPrivileges, "%s", actionErr)
	case strings.Contains(e, "connection") || strings.Contains(e, "dial") || strings.Contains(e, "unreachable") || strings.Contains(e, "no such host"):
		return newToolError(codeAgentUnreachable, "%s", actionErr)
	case strings.Contains(e, "1064") || strings.Contains(e, "syntax"):
		// EXPLAIN ran on the ? fingerprint because no concrete statement was
		// available: needs a stored query example or explicit placeholders.
		return newToolError(codeInvalidInput,
			"EXPLAIN could not run on the fingerprint (placeholder syntax): %s. "+
				"Enable query examples for this service in PMM, or pass placeholders (one value per ?, in order) and database.", actionErr)
	case strings.Contains(e, "not found") || strings.Contains(e, "doesn't exist") || strings.Contains(e, "does not exist"):
		return newToolError(codeNotFound, "%s", actionErr)
	default:
		return newToolError(codePMMUnavailable, "%s", actionErr)
	}
}

// decodeExplainOutput unwraps the MySQL explain envelope
// {explain_result: <base64 plan>, explained_query, is_dml} (source-confirmed in
// agent/runner/actions/mysql_explain_action.go). With raw SQL disabled the
// explained statement (a real query with literals) is omitted. The plan body
// itself may still embed literals (e.g. attached_condition in FORMAT=JSON):
// disable query examples in PMM to prevent literal-bearing EXPLAINs entirely.
func decodeExplainOutput(output string, rawSQL bool) string {
	var envelope struct {
		ExplainResult  string `json:"explain_result"`
		ExplainedQuery string `json:"explained_query"`
		IsDML          bool   `json:"is_dml"`
	}
	err := json.Unmarshal([]byte(output), &envelope)
	if err != nil || envelope.ExplainResult == "" {
		return output
	}
	plan, err := base64.StdEncoding.DecodeString(envelope.ExplainResult)
	if err != nil {
		return output
	}
	text := strings.TrimRight(string(plan), "\n")
	if rawSQL && envelope.ExplainedQuery != "" {
		prefix := "-- " + envelope.ExplainedQuery
		if envelope.IsDML {
			prefix += "\n-- (DML statement rewritten to an equivalent SELECT by pmm-agent)"
		}
		return prefix + "\n" + text
	}
	return text
}
