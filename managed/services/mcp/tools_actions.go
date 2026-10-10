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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

const (
	// DefaultActionTimeout bounds one pmm_get_explain or pmm_get_schema call.
	DefaultActionTimeout = 15 * time.Second

	pollInitialDelay = 300 * time.Millisecond
	pollMaxDelay     = 2 * time.Second
	pollBackoff      = 1.5

	formatJSON        = "json"
	formatTraditional = "traditional"

	infoDefinition = "definition"
	infoIndexes    = "indexes"
	infoAll        = "all"

	// Tables per pmm_get_schema call, at most.
	maxSchemaTables = 10
	// Actions in flight per call, at most, so that one call cannot flood a pmm-agent.
	maxActionsInFlight = 4
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
	ServiceID string   `json:"service_id" jsonschema:"Service id from pmm_inventory"`
	Database  string   `json:"database" jsonschema:"Database / schema name"`
	Tables    []string `json:"tables" jsonschema:"1 to 10 table names"`
	InfoTypes string   `json:"info_types,omitempty" jsonschema:"definition (default), indexes or all"`
}

func (s *Service) registerActionTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_get_explain",
		Title: "Execution plan",
		Description: "Return the execution plan for a query addressed by queryid. PostgreSQL plans come from " +
			"pg_stat_monitor's stored plan (needs pgsm_enable_query_plan=on). MySQL plans are produced live by " +
			"the PMM agent with a non-executing EXPLAIN (never EXPLAIN ANALYZE). MongoDB plans are produced live " +
			"with explain, which runs the query's candidate plans to collect execution statistics but never " +
			"modifies data. If a MySQL fingerprint has ? placeholders and no query example is stored, pass " +
			"placeholders and database.",
		Annotations: readOnly("Execution plan"),
	}, handle(s, "pmm_get_explain", s.explain))

	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_get_schema",
		Title: "Table DDL",
		Description: "Return SHOW CREATE TABLE (MySQL or PostgreSQL) and/or the indexes of up to 10 tables via the PMM agent, " +
			"fetched in parallel. A table that fails shows its error inline; the others still return.",
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
	err := checkQANValue("queryid", in.QueryID)
	if err != nil {
		return nil, err
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
	// One PMM_MCP_ACTION_TIMEOUT bounds the whole call, lookups and actions alike.
	ctx, cancel := context.WithTimeout(ctx, s.actionTimeout())
	defer cancel()
	// Read once: each read is a settings query, and two reads could disagree.
	raw := s.rawSQL()

	// Resolve the service before anything else: the engine decides where the
	// plan comes from, and an unknown service_id must fail before any plan is
	// fetched.
	svc, err := s.engineOf(ctx, auth, in.ServiceID)
	if err != nil {
		return nil, err
	}

	// PostgreSQL has no live EXPLAIN action; its plans come only from
	// pg_stat_monitor. MySQL and MongoDB never have stored plans, so they skip
	// the probe rather than paying for a lookup that cannot succeed.
	if svc.Engine == enginePostgreSQL {
		return s.storedPlan(ctx, auth, in.QueryID, svc, base, now, raw)
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
	default:
		return nil, newToolError(codeInvalidInput, "EXPLAIN is not available for engine '%s'", svc.Engine)
	}

	output, err := s.runAction(ctx, auth, body, !raw)
	var te *toolError
	if errors.As(err, &te) && te.code == codeInvalidInput && strings.HasPrefix(te.message, "Error 1064 ") {
		// MySQL answered 1064: EXPLAIN ran on the ? fingerprint because no
		// concrete statement was available.
		return nil, newToolError(codeInvalidInput,
			"EXPLAIN could not run on the fingerprint (placeholder syntax): %s. "+
				"Enable query examples for this service in PMM, or pass placeholders (one value per ?, in order) and database.", te.message)
	}
	if err != nil {
		return nil, err
	}

	plan, decoded := decodeExplainOutput(output, raw)
	if !raw {
		plan = redactPlan(svc.Engine, format, plan, decoded)
	}
	text := fmt.Sprintf("EXPLAIN (%s, %s %s):\n```\n%s\n```", format, svc.Engine, svc.ServiceName, plan)
	if in.QueryID != "" {
		text += "\n\n" + link("View this query in PMM", qanQueryURL(base, in.QueryID, svc.ServiceName, now.Add(-time.Hour), now, "explain"))
	}
	return textResult(text), nil
}

// storedPlan returns pg_stat_monitor's stored plan for a digest, or none on 200 {};
// qan-api2 ignores the service_id it is sent until PMM-15697.
func (s *Service) storedPlan(
	ctx context.Context, auth callerAuth, queryID string, svc serviceInfo, base string, now time.Time, raw bool,
) (*mcp.CallToolResult, error) {
	if queryID == "" {
		return nil, newToolError(codeInvalidInput, "PostgreSQL EXPLAIN is addressed by queryid; pass the queryid from pmm_top_queries")
	}

	plan, err := s.api.GetQueryPlan(ctx, auth, queryID, svc.ServiceID)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.QueryPlan == "" {
		return nil, newToolError(codeNotFound,
			"no stored plan for queryid '%s'. PMM has no live EXPLAIN action for PostgreSQL, so plans come only from "+
				"pg_stat_monitor: monitor the service with pg_stat_monitor and pg_stat_monitor.pgsm_enable_query_plan=on, "+
				"then retry once the query has run again", queryID)
	}

	body := plan.QueryPlan
	if !raw {
		body = maskPGPlan(body)
	}
	text := "Stored plan (pg_stat_monitor, planid " + plan.Planid + "):\n```\n" + body + "\n```"
	text += "\n\n" + link("View this query in PMM", qanQueryURL(base, queryID, svc.ServiceName, now.Add(-time.Hour), now, "explain"))
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
	infoTypes := cmp.Or(in.InfoTypes, infoDefinition)
	if !slices.Contains([]string{infoDefinition, infoIndexes, infoAll}, infoTypes) {
		return nil, newToolError(codeInvalidInput, "info_types must be definition, indexes or all; got '%s'", in.InfoTypes)
	}
	tables, err := checkTables(in.Tables)
	if err != nil {
		return nil, err
	}
	if in.ServiceID == "" || in.Database == "" {
		return nil, newToolError(codeInvalidInput, "service_id and database are required")
	}
	auth := callerAuthFromHeader(req.Extra.Header)
	ctx, cancel := context.WithTimeout(ctx, s.actionTimeout())
	defer cancel()

	svc, err := s.engineOf(ctx, auth, in.ServiceID)
	if err != nil {
		return nil, err
	}
	refs := make([]tableRef, len(tables))
	for i, t := range tables {
		refs[i] = tableRef{database: in.Database, name: t}
	}
	return s.tableInfo(ctx, auth, svc, refs, infoTypes)
}

// tableRef names one table.
type tableRef struct {
	database string
	name     string
}

// checkTables validates the tables input and drops repeats, keeping the order.
func checkTables(tables []string) ([]string, error) {
	out := make([]string, 0, len(tables))
	for _, t := range tables {
		if strings.TrimSpace(t) == "" {
			return nil, newToolError(codeInvalidInput, "tables must not contain an empty name")
		}
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	if len(out) == 0 || len(out) > maxSchemaTables {
		return nil, newToolError(codeInvalidInput, "tables takes 1 to %d names; got %d", maxSchemaTables, len(out))
	}
	return out, nil
}

// schemaJob is one action of a pmm_get_schema call.
type schemaJob struct {
	// table is an index into the call's tables.
	table   int
	indexes bool
	body    actions_service.StartServiceActionBody
}

// tableInfo runs the DDL and index actions of tables, at most maxActionsInFlight
// at a time and all under ctx's one deadline, and renders a block per table in
// input order. It fails only when every action fails.
func (s *Service) tableInfo(ctx context.Context, auth callerAuth, svc serviceInfo, tables []tableRef, infoTypes string) (*mcp.CallToolResult, error) {
	if svc.Engine != engineMySQL && svc.Engine != enginePostgreSQL {
		return nil, newToolError(codeInvalidInput, "SHOW CREATE TABLE is not available for engine '%s'", svc.Engine)
	}
	var jobs []schemaJob
	for i, t := range tables {
		ddl, idx := schemaBodies(svc, t)
		if infoTypes != infoIndexes {
			jobs = append(jobs, schemaJob{table: i, body: ddl})
		}
		if infoTypes != infoDefinition {
			jobs = append(jobs, schemaJob{table: i, indexes: true, body: idx})
		}
	}

	outputs := make([]string, len(jobs))
	errs := make([]error, len(jobs))
	slots := make(chan struct{}, maxActionsInFlight)
	var wg sync.WaitGroup
	for i, job := range jobs {
		wg.Go(func() {
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
				// SHOW CREATE TABLE and SHOW INDEX errors quote only the caller's
				// identifiers, never a value, so they are not masked.
				outputs[i], errs[i] = s.runAction(ctx, auth, job.body, false)
			case <-ctx.Done():
				errs[i] = s.deadlineOr(ctx, "", ctx.Err())
			}
		})
	}
	wg.Wait()
	if !slices.Contains(errs, nil) {
		return nil, errs[0]
	}

	blocks := make([]string, len(tables))
	for i, t := range tables {
		blocks[i] = "### " + t.database + "." + t.name
	}
	for i, job := range jobs {
		blocks[job.table] += "\n" + schemaPart(job.indexes, outputs[i], errs[i])
	}
	return textResult(strings.Join(blocks, "\n\n")), nil
}

// schemaBodies returns the DDL and index action bodies of a table.
func schemaBodies(svc serviceInfo, t tableRef) (actions_service.StartServiceActionBody, actions_service.StartServiceActionBody) {
	var ddl, idx actions_service.StartServiceActionBody
	if svc.Engine == engineMySQL {
		ddl.MysqlShowCreateTable = &actions_service.StartServiceActionParamsBodyMysqlShowCreateTable{
			ServiceID: svc.ServiceID, Database: t.database, TableName: t.name,
		}
		idx.MysqlShowIndex = &actions_service.StartServiceActionParamsBodyMysqlShowIndex{
			ServiceID: svc.ServiceID, Database: t.database, TableName: t.name,
		}
		return ddl, idx
	}
	ddl.PostgresShowCreateTable = &actions_service.StartServiceActionParamsBodyPostgresShowCreateTable{
		ServiceID: svc.ServiceID, Database: t.database, TableName: t.name,
	}
	idx.PostgresShowIndex = &actions_service.StartServiceActionParamsBodyPostgresShowIndex{
		ServiceID: svc.ServiceID, Database: t.database, TableName: t.name,
	}
	return ddl, idx
}

// schemaPart renders one action's result inside its table's block.
func schemaPart(indexes bool, output string, err error) string {
	switch {
	case err != nil:
		return mapError(err).Error()
	case indexes:
		return "indexes:\n```\n" + strings.TrimRight(output, "\n") + "\n```"
	case strings.TrimSpace(output) == "":
		return "No DDL returned for that table."
	default:
		return "```sql\n" + strings.TrimRight(output, "\n") + "\n```"
	}
}

// runAction starts a service action and polls it with 300 ms -> 2 s backoff
// until it is done or ctx ends; ctx carries the tool call's one action
// deadline. With redact set, literals in the action's error are masked.
func (s *Service) runAction(ctx context.Context, auth callerAuth, body actions_service.StartServiceActionBody, redact bool) (string, error) {
	started, err := s.api.StartServiceAction(ctx, auth, body)
	if err != nil {
		return "", s.deadlineOr(ctx, "", mapActionStartError(err))
	}
	actionID := actionIDOf(started)
	if actionID == "" {
		return "", newToolError(codePMMUnavailable, "no action_id returned by startServiceAction")
	}

	delay := pollInitialDelay
	for {
		res, err := s.api.GetAction(ctx, auth, actionID)
		if err != nil {
			return "", s.deadlineOr(ctx, actionID, err)
		}
		if res.Done {
			if res.Error != "" {
				return "", mapActionError(res.Error, redact)
			}
			return res.Output, nil
		}
		select {
		case <-ctx.Done():
			return "", s.deadlineOr(ctx, actionID, ctx.Err())
		case <-time.After(delay):
		}
		delay = min(time.Duration(float64(delay)*pollBackoff), pollMaxDelay)
	}
}

// deadlineOr returns the timeout error once ctx's deadline has passed, else err.
func (s *Service) deadlineOr(ctx context.Context, actionID string, err error) error {
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return err
	}
	if actionID == "" {
		return newToolError(codeTimeout, "the action did not start within %s (PMM_MCP_ACTION_TIMEOUT)", s.actionTimeout())
	}
	return newToolError(codeTimeout, "action %s did not complete within %s (PMM_MCP_ACTION_TIMEOUT)", actionID, s.actionTimeout())
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

// mysqlErrors classifies MySQL error numbers. Those marked identifiers quote
// only users, hosts, databases, tables, keys and routines, never a value, so
// their message is kept when raw SQL is off; remediation needs the names.
var mysqlErrors = map[string]struct {
	code        errorCode
	identifiers bool
}{
	"1044": {codeInsufficientPrivileges, true},
	"1045": {codeInsufficientPrivileges, true},
	"1142": {codeInsufficientPrivileges, true},
	"1143": {codeInsufficientPrivileges, true},
	"1227": {codeInsufficientPrivileges, true},
	"1370": {codeInsufficientPrivileges, true},
	"1698": {codeInsufficientPrivileges, true},
	"1040": {codeAgentUnreachable, true},
	"1203": {codeAgentUnreachable, true},
	"1049": {codeNotFound, true},
	"1146": {codeNotFound, true},
	"1176": {codeNotFound, true},
	"1305": {codeNotFound, true},
	"1449": {codeNotFound, true},
	"1932": {codeNotFound, true},
	// ANSI_QUOTES makes a double-quoted string a column name, so 1054 can
	// quote a value.
	"1054": {codeNotFound, false},
	"1064": {codeInvalidInput, false},
}

// mongoErrors classifies MongoDB server error code names.
var mongoErrors = map[string]errorCode{
	"Unauthorized":         codeInsufficientPrivileges,
	"AuthenticationFailed": codeInsufficientPrivileges,
	"NamespaceNotFound":    codeNotFound,
	"IndexNotFound":        codeNotFound,
}

// withheldMessage replaces an action error message that can quote the statement.
const withheldMessage = "(message withheld: it can quote the statement, and PMM_MCP_RAW_SQL is off)"

// mapActionError classifies the error text of a finished action.
//
// With redact set (raw SQL off), the message is withheld unless it is known
// not to quote the statement the agent ran, with its literal values:
// pmm-agent's own fixed texts and network errors (safeActionError), and the
// MySQL errors that name only identifiers. Every other shape - MySQL's
// "near '...'", MongoDB's planner and parse errors, a format not seen yet -
// is withheld, keeping its MySQL error number or MongoDB code name.
//
// The error is classified by that number or name, and otherwise on its text
// with the quoted parts removed, so that a word inside a quoted value cannot
// pass for the error's own.
func mapActionError(actionErr string, redact bool) *toolError {
	rest := strings.TrimPrefix(actionErr, mongoExplainPrefix)
	prefix := actionErr[:len(actionErr)-len(rest)]
	mysqlHeader := mysqlErrorHeader.FindStringSubmatch(rest)
	mongoName := mongoErrorName.FindStringSubmatch(rest)

	var code errorCode
	safe := safeActionError.MatchString(rest)
	header := ""
	switch {
	case mysqlHeader != nil:
		known := mysqlErrors[mysqlHeader[1]]
		code, safe, header = known.code, known.identifiers, mysqlHeader[0]
	case mongoName != nil:
		code, header = mongoErrors[mongoName[1]], mongoName[0]
	}
	if code == "" {
		code = classifyErrorText(maskQuoted(rest))
	}

	msg := actionErr
	if redact && !safe {
		msg = prefix + header + withheldMessage
	}
	return newToolError(code, "%s", msg)
}

// classifyErrorText classifies an error by the words of its message.
func classifyErrorText(text string) errorCode {
	e := strings.ToLower(text)
	switch {
	case strings.Contains(e, "denied") || strings.Contains(e, "privilege") || strings.Contains(e, "permission") ||
		strings.Contains(e, "not authorized") || strings.Contains(e, "authentication failed"):
		return codeInsufficientPrivileges
	case strings.Contains(e, "connection") || strings.Contains(e, "dial") || strings.Contains(e, "unreachable") ||
		strings.Contains(e, "no such host") || strings.Contains(e, "server selection"):
		return codeAgentUnreachable
	case strings.Contains(e, "not found") || strings.Contains(e, "doesn't exist") || strings.Contains(e, "does not exist"):
		return codeNotFound
	default:
		return codePMMUnavailable
	}
}

// decodeExplainOutput unwraps the MySQL explain envelope
// {explain_result: <base64 plan>, explained_query, is_dml} (source-confirmed in
// agent/runner/actions/mysql_explain_action.go). With raw SQL disabled the
// explained statement (a real query with literals) is omitted; literals in the
// plan body are masked separately, by redactPlan. It reports whether output
// was such an envelope: anything else is returned unchanged.
func decodeExplainOutput(output string, rawSQL bool) (string, bool) {
	var envelope struct {
		ExplainResult  string `json:"explain_result"`
		ExplainedQuery string `json:"explained_query"`
		IsDML          bool   `json:"is_dml"`
	}
	err := json.Unmarshal([]byte(output), &envelope)
	if err != nil || envelope.ExplainResult == "" {
		return output, false
	}
	plan, err := base64.StdEncoding.DecodeString(envelope.ExplainResult)
	if err != nil {
		return output, false
	}
	text := strings.TrimRight(string(plan), "\n")
	if rawSQL && envelope.ExplainedQuery != "" {
		prefix := "-- " + envelope.ExplainedQuery
		if envelope.IsDML {
			prefix += "\n-- (DML statement rewritten to an equivalent SELECT by pmm-agent)"
		}
		return prefix + "\n" + text, true
	}
	return text, true
}
