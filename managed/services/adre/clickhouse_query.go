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
	tableRefPattern = regexp.MustCompile(`(?is)\b(?:FROM|JOIN)\s+(?:ONLY\s+)?([a-zA-Z][a-zA-Z0-9_.]*)`)
	limitPattern    = regexp.MustCompile(`(?is)\blimit\s+(\d+)\s*(?:offset\s+\d+)?\s*$`)
	// LLM often emits JSON-style map keys; ClickHouse requires single-quoted keys in ['key'].
	clickHouseMapDoubleQuoteKey = regexp.MustCompile(`(?i)(ResourceAttributes|LogAttributes|ScopeAttributes|InstrumentationScopeAttributes)\["([^"]+)"\]`)
	// Holmes bash escaping for jq --arg / "{{ query }}" becomes '"key"' or '"'"' in SQL.
	clickHouseHolmesQuotedSingleton = regexp.MustCompile(`'"([^']+)"'`)
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
	Database     string   `json:"database"`
	Columns      []string `json:"columns"`
	Rows         [][]any  `json:"rows"`
	RowCount     int      `json:"row_count"`
	Truncated    bool     `json:"truncated"`
	ExecutionMS  int64    `json:"execution_ms"`
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
	if err := json.Unmarshal(body, &req); err != nil {
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
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) clickHouseDB(database string) (*sql.DB, bool) {
	switch strings.ToLower(strings.TrimSpace(database)) {
	case "pmm":
		if h.clickhouse.PMM == nil {
			return nil, false
		}
		return h.clickhouse.PMM, true
	case "otel":
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

// validateClickHouseQuery applies guardrails and returns the query to execute (with LIMIT enforced).
func validateClickHouseQuery(database, query string, maxRows int) (string, error) {
	db := strings.ToLower(strings.TrimSpace(database))
	if db != "pmm" && db != "otel" {
		return "", fmt.Errorf("database must be pmm or otel")
	}

	q := normalizeClickHouseQuerySQL(query)
	q = strings.TrimSuffix(q, ";")
	q = strings.TrimSpace(q)
	if q == "" {
		return "", fmt.Errorf("query is empty")
	}
	if strings.Contains(q, ";") {
		return "", fmt.Errorf("multiple statements are not allowed")
	}

	upper := strings.ToUpper(q)
	for _, kw := range forbiddenSQLKeywords {
		if regexp.MustCompile(`\b` + kw + `\b`).MatchString(upper) {
			return "", fmt.Errorf("forbidden keyword %s", kw)
		}
	}
	for _, pat := range forbiddenSQLPatterns {
		if pat.MatchString(q) {
			return "", fmt.Errorf("forbidden SQL pattern")
		}
	}

	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return "", fmt.Errorf("only SELECT queries are allowed")
	}
	if strings.HasPrefix(upper, "WITH") && !regexp.MustCompile(`(?i)\bSELECT\b`).MatchString(q) {
		return "", fmt.Errorf("WITH queries must contain SELECT")
	}

	tables, err := extractClickHouseTables(q)
	if err != nil {
		return "", err
	}
	if len(tables) == 0 {
		return "", fmt.Errorf("query must reference a table in FROM or JOIN")
	}
	for _, t := range tables {
		if err := validateClickHouseTable(db, t); err != nil {
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
	return strings.TrimSpace(q)
}

func extractClickHouseTables(query string) ([]string, error) {
	matches := tableRefPattern.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(m[1]))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
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
	if m := limitPattern.FindStringSubmatch(query); len(m) == 2 {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return "", fmt.Errorf("invalid LIMIT value")
		}
		if n > maxRows {
			return "", fmt.Errorf("LIMIT %d exceeds max_rows %d", n, maxRows)
		}
		return query, nil
	}
	return strings.TrimSpace(query) + fmt.Sprintf(" LIMIT %d", maxRows), nil
}
