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

package adre

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/percona/pmm/utils/sqlrows"
)

const (
	defaultClickHouseMaxRows = 500
	hardClickHouseMaxRows    = 1000
	clickHouseQueryTimeout   = 30 * time.Second
)

var (
	forbiddenSQLKeywords = []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "TRUNCATE",
		"SYSTEM", "GRANT", "REVOKE", "ATTACH", "DETACH", "RENAME", "OPTIMIZE",
		"KILL", "EXCHANGE",
	}
	forbiddenSQLPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bINTO\s+OUTFILE\b`),
		regexp.MustCompile(`(?i)\bFORMAT\s+\w+`), // e.g. FORMAT JSON — not the format() function
		regexp.MustCompile(`(?i)\bREPLACE\s+TABLE\b`),
	}
	// Captures the source list after FROM/JOIN (up to the next clause boundary, a
	// parenthesis, or end of statement) so EVERY comma-separated source — not just
	// the first — is checked against the table allowlist. A table function such as
	// url is captured as the bare name "url", which is not an allowlisted table and
	// is therefore rejected. A subquery begins with a parenthesis so it does not
	// match here; its inner FROM/JOIN tables are matched on their own.
	fromSourcePattern = regexp.MustCompile(`(?is)\b(?:FROM|JOIN)\s+(?:ONLY\s+)?([a-zA-Z_][^()]*?)\s*` +
		`(?:\(|\)|,?\s*$|\b(?:WHERE|PREWHERE|GROUP|ORDER|LIMIT|HAVING|SETTINGS|UNION|FORMAT|` +
		`ON|USING|SAMPLE|FINAL|ARRAY|LEFT|RIGHT|INNER|OUTER|FULL|CROSS|GLOBAL|ANY|ALL|ASOF|SEMI|ANTI|JOIN)\b)`)
	sourceIdentPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*`)
	// Rejects ClickHouse table/remote/file functions and the scalar dictionary/join readers anywhere
	// in the statement (projection, scalar subquery, comma cross-join, or a no-space FROM(fn(...))).
	// They can read arbitrary files, reach arbitrary hosts (SSRF), run a script (executable), or read
	// data outside the metrics/logs allowlist (including dictGet/joinGet, which are scalar and run even
	// under readonly=1), and are never needed for metrics/logs analytics. This denylist must be kept in
	// sync with new ClickHouse functions; the fromSourcePattern positive allowlist is the backstop.
	forbiddenTableFunctionPattern = regexp.MustCompile(`(?i)\b(?:` +
		`url|urlCluster|file|fileCluster|s3|s3Cluster|s3queue|gcs|gcsCluster|` +
		`remote|remoteSecure|cluster|clusterAllReplicas|` +
		`mysql|postgresql|mongodb|redis|sqlite|jdbc|odbc|` +
		`hdfs|hdfsCluster|azureBlobStorage|azureBlobStorageCluster|` +
		`deltaLake|deltaLakeCluster|hudi|hudiCluster|iceberg|icebergS3|icebergAzure|icebergHDFS|icebergCluster|` +
		`merge|mergeTreeIndex|mergeTreeProjection|input|dictionary|view|viewIfPermitted|loop|executable|` +
		`dictGet\w*|joinGet\w*` +
		`)\s*\(`)
	limitPattern = regexp.MustCompile(`(?is)\blimit\s+(\d+)\s*(?:offset\s+\d+)?\s*$`)
	// LLM often emits JSON-style map keys; ClickHouse requires single-quoted keys in ['key'].
	clickHouseMapDoubleQuoteKey = regexp.MustCompile(`(?i)(ResourceAttributes|LogAttributes|ScopeAttributes|InstrumentationScopeAttributes)\["([^"]+)"\]`)
	// Holmes bash escaping for jq --arg / "{{ query }}" becomes '"key"' or '"'"' in SQL.
	clickHouseHolmesQuotedSingleton = regexp.MustCompile(`'"([^']+)"'`)
	// LLMs (esp. Non-frontier models) sometimes drop the space in datetime literals,
	// e.g. '2026-06-0408:10:08' instead of '2026-06-04 08:10:08', which ClickHouse rejects
	// with "Cannot convert string ... To type DateTime" (code 53). There is no valid format
	// where YYYY-MM-DD is immediately followed by HH:MM:SS, so re-inserting the space is safe.
	clickHouseDateTimeMissingSpace = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})(\d{2}:\d{2}:\d{2})`)
)

// ClickHousePools holds native-protocol connections for ADRE read-only queries.
type ClickHousePools struct {
	PMM  *sql.DB
	OTel *sql.DB
}

type clickHouseQueryRequest struct {
	Database string `json:"database"`
	Query    string `json:"query"`
	MaxRows  int    `json:"max_rows"`
}

type clickHouseQueryResponse struct {
	Database    string   `json:"database"`
	Columns     []string `json:"columns"`
	Rows        [][]any  `json:"rows"`
	RowCount    int      `json:"row_count"`
	Truncated   bool     `json:"truncated"`
	ExecutionMS int64    `json:"execution_ms"`
}

// PostClickHouseQuery handles POST /v1/adre/clickhouse/query.
func (h *Handlers) PostClickHouseQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) //nolint:mnd
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req clickHouseQueryRequest
	if err := json.Unmarshal(body, &req); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if _, ok := h.checkAdreEnabled(w); !ok {
		return
	}

	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = defaultClickHouseMaxRows
	}
	if maxRows > hardClickHouseMaxRows {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("max_rows exceeds hard cap (%d)", hardClickHouseMaxRows))
		return
	}

	prepared, err := validateClickHouseQuery(req.Database, req.Query, maxRows)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	db, ok := h.clickHouseDB(req.Database)
	if !ok {
		writeJSONError(w, http.StatusServiceUnavailable, "clickhouse not configured for "+req.Database)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), clickHouseQueryTimeout)
	defer cancel()

	start := time.Now()
	rows, err := db.QueryContext(ctx, prepared)
	if err != nil {
		h.l.Warnf("clickhouse query failed database=%s query_sha=%s user=%q err=%v",
			req.Database, queryFingerprint(req.Query), h.userLoginFromRequest(r), err)
		writeJSONError(w, http.StatusBadGateway, "clickhouse query failed: "+err.Error())
		return
	}

	columns, dataRows, err := sqlrows.ReadRows(rows)
	if err != nil {
		h.l.Warnf("clickhouse read rows database=%s query_sha=%s err=%v", req.Database, queryFingerprint(req.Query), err)
		writeJSONError(w, http.StatusBadGateway, "failed to read clickhouse result")
		return
	}

	truncated := len(dataRows) > maxRows
	if truncated {
		dataRows = dataRows[:maxRows]
	}

	elapsed := time.Since(start).Milliseconds()
	h.l.Infof("clickhouse query database=%s rows=%d truncated=%v ms=%d query_sha=%s user=%q",
		req.Database, len(dataRows), truncated, elapsed, queryFingerprint(req.Query), h.userLoginFromRequest(r))

	resp := clickHouseQueryResponse{
		Database:    req.Database,
		Columns:     columns,
		Rows:        dataRows,
		RowCount:    len(dataRows),
		Truncated:   truncated,
		ExecutionMS: elapsed,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp) //nolint:errchkjson
}

func (h *Handlers) clickHouseDB(database string) (*sql.DB, bool) {
	switch strings.ToLower(strings.TrimSpace(database)) {
	case "pmm": //nolint:goconst
		if h.clickhouse.PMM == nil {
			return nil, false
		}
		return h.clickhouse.PMM, true
	case "otel": //nolint:goconst
		if h.clickhouse.OTel == nil {
			return nil, false
		}
		return h.clickhouse.OTel, true
	default:
		return nil, false
	}
}

func queryFingerprint(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:8])
}

// maskClickHouseQueryForScan returns a copy of q for guard scanning: single-quoted string literals are
// emptied ('...' -> ”) and SQL comments (/* */ and -- to end of line) are replaced with a space, in a
// single quote-aware pass. Doing both together (rather than two regex passes) stops a comment delimiter
// inside a literal — or a quote inside a comment — from confusing the guards, and stops the forbidden
// function denylist from being evaded with a comment between the name and "(" (e.g. url/**/()): the
// comment collapses to a space so url/**/( becomes "url (", which the denylist still matches.
func maskClickHouseQueryForScan(q string) string {
	var b strings.Builder
	b.Grow(len(q))
	r := []rune(q)
	for i := 0; i < len(r); i++ {
		switch {
		case r[i] == '\'': // string literal: emit '' and consume to the closing quote ('' = escaped quote)
			b.WriteString("''")
			i++
			for i < len(r) {
				if r[i] == '\'' {
					if i+1 < len(r) && r[i+1] == '\'' {
						i += 2
						continue
					}
					break // closing quote (outer loop's i++ steps past it)
				}
				i++
			}
		case r[i] == '-' && i+1 < len(r) && r[i+1] == '-': // line comment to end of line
			b.WriteByte(' ')
			for i < len(r) && r[i] != '\n' {
				i++
			}
			if i < len(r) {
				b.WriteByte('\n') // preserve the line break (outer loop's i++ steps past it)
			}
		case r[i] == '/' && i+1 < len(r) && r[i+1] == '*': // block comment
			b.WriteByte(' ')
			i += 2
			for i+1 < len(r) && !(r[i] == '*' && r[i+1] == '/') {
				i++
			}
			i++ // skip the closing '*' (outer loop's i++ steps past the '/')
		default:
			b.WriteRune(r[i])
		}
	}
	return b.String()
}

// validateClickHouseQuery applies guardrails and returns the query to execute (with LIMIT enforced).
func validateClickHouseQuery(database, query string, maxRows int) (string, error) {
	db := strings.ToLower(strings.TrimSpace(database))
	if db != "pmm" && db != "otel" {
		return "", errors.New("database must be pmm or otel")
	}

	q := normalizeClickHouseQuerySQL(query)
	q = strings.TrimSuffix(q, ";")
	q = strings.TrimSpace(q)
	if q == "" {
		return "", errors.New("query is empty")
	}
	if strings.Contains(q, ";") {
		return "", errors.New("multiple statements are not allowed")
	}

	// Empty string literals and strip comments before scanning so (a) a forbidden keyword/function/table
	// that appears only inside a literal (e.g. WHERE Body LIKE '%drop%' or '%url(%') is not mistaken for
	// SQL, and (b) a comment between a function name and its "(" (e.g. url/**/()) cannot evade the
	// denylist. The executed query (q) keeps its real literals; ClickHouse treats comments as whitespace.
	masked := maskClickHouseQueryForScan(q)
	upper := strings.ToUpper(masked)
	for _, kw := range forbiddenSQLKeywords {
		if regexp.MustCompile(`\b` + kw + `\b`).MatchString(upper) {
			return "", fmt.Errorf("forbidden keyword %s", kw)
		}
	}
	for _, pat := range forbiddenSQLPatterns {
		if pat.MatchString(masked) {
			return "", errors.New("forbidden SQL pattern")
		}
	}

	if forbiddenTableFunctionPattern.MatchString(masked) {
		return "", errors.New("table functions are not allowed")
	}

	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return "", errors.New("only SELECT queries are allowed")
	}
	if strings.HasPrefix(upper, "WITH") && !regexp.MustCompile(`(?i)\bSELECT\b`).MatchString(q) {
		return "", errors.New("WITH queries must contain SELECT")
	}

	tables, err := extractClickHouseTables(q)
	if err != nil {
		return "", err
	}
	if len(tables) == 0 {
		return "", errors.New("query must reference a table in FROM or JOIN")
	}
	for _, t := range tables {
		err := validateClickHouseTable(db, t)
		if err != nil {
			return "", err
		}
	}

	q, err = enforceClickHouseLimit(q, maxRows)
	if err != nil {
		return "", err
	}

	return q + " SETTINGS max_execution_time=30, readonly=1", nil
}

// trimClickHouseQueryQuotes removes one or more layers of surrounding ' or " the LLM may add.
func trimClickHouseQueryQuotes(q string) string {
	q = strings.TrimSpace(q)
	for len(q) >= 2 {
		start, end := q[0], q[len(q)-1]
		if (start == '\'' && end == '\'') || (start == '"' && end == '"') {
			q = strings.TrimSpace(q[1 : len(q)-1])
			continue
		}
		break
	}
	return q
}

// normalizeClickHouseQuerySQL fixes common LLM/JSON quoting before ClickHouse executes the query.
func normalizeClickHouseQuerySQL(query string) string {
	q := trimClickHouseQueryQuotes(strings.TrimSpace(query))
	// Holmes/JSON sometimes sends backslash-escaped quotes; ClickHouse rejects \' (syntax error 62).
	q = strings.ReplaceAll(q, `\'`, `'`)
	q = strings.ReplaceAll(q, `\"`, `"`)
	// Holmes shell quote-joining for jq --arg: collapse '"'"' to '.
	q = strings.ReplaceAll(q, `'"'"'`, `'`)
	// Holmes mangling: '"node_name"' -> 'node_name' (wrong quotes, often returns 0 rows without syntax error).
	q = clickHouseHolmesQuotedSingleton.ReplaceAllString(q, "'$1'")
	// JSON-style map keys ["key"] are parsed as identifiers (error 47); rewrite to ['key'].
	q = clickHouseMapDoubleQuoteKey.ReplaceAllString(q, "$1['$2']")
	// Repair datetime literals missing the date/time space (error 53). Safe: no valid format joins them.
	q = clickHouseDateTimeMissingSpace.ReplaceAllString(q, "$1 $2")
	return strings.TrimSpace(q)
}

func extractClickHouseTables(query string) ([]string, error) { //nolint:unparam
	masked := maskClickHouseQueryForScan(query)
	matches := fromSourcePattern.FindAllStringSubmatch(masked, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, m := range matches {
		if len(m) < 2 { //nolint:mnd
			continue
		}
		for part := range strings.SplitSeq(m[1], ",") {
			name := strings.ToLower(sourceIdentPattern.FindString(strings.TrimSpace(part)))
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out, nil
}

func validateClickHouseTable(database, table string) error {
	switch database {
	case "pmm":
		switch table {
		case "metrics", "pmm.metrics":
			return nil
		default:
			return fmt.Errorf("table %q not allowed for database pmm (use metrics or pmm.metrics)", table)
		}
	case "otel":
		switch table {
		case "logs", "otel.logs":
			return nil
		default:
			return fmt.Errorf("table %q not allowed for database otel (use logs or otel.logs)", table)
		}
	default:
		return fmt.Errorf("unknown database %q", database)
	}
}

func enforceClickHouseLimit(query string, maxRows int) (string, error) {
	if m := limitPattern.FindStringSubmatch(query); len(m) == 2 { //nolint:mnd
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return "", errors.New("invalid LIMIT value")
		}
		if n > maxRows {
			return "", fmt.Errorf("LIMIT %d exceeds max_rows %d", n, maxRows)
		}
		return query, nil
	}
	return strings.TrimSpace(query) + fmt.Sprintf(" LIMIT %d", maxRows), nil
}
