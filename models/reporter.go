package models

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
)

// Reporter implements models to select query classes by params.
type Reporter struct {
	db *sqlx.DB
}

// NewReporter initialize Reporter with db instance.
func NewReporter(db *sqlx.DB) Reporter {
	return Reporter{db: db}
}

var funcMap = template.FuncMap{
	"inc":         func(i int) int { return i + 1 },
	"StringsJoin": strings.Join,
}

const queryReportTotalTmpl = `
SELECT
SUM(num_queries) AS num_queries,
SUM(m_query_time_cnt) AS m_query_time_cnt,
SUM(m_query_time_sum) AS m_query_time_sum,
MIN(m_query_time_min) AS m_query_time_min,
MAX(m_query_time_max) AS m_query_time_max,
AVG(m_query_time_p99) AS m_query_time_p99
FROM queries
WHERE period_start > :from AND period_start < :to
{{ if index . "servers" }} AND db_server IN ( :servers ) {{ end }}
{{ if index . "schemas" }} AND db_schema IN ( :schemas ) {{ end }}
{{ if index . "users" }} AND db_username IN ( :users ) {{ end }}
{{ if index . "hosts" }} AND client_host IN ( :hosts ) {{ end }}
{{ if index . "labels" }}
AND (
		{{$i := 0}}
		{{range $key, $val := index . "labels"}}
			{{ $i = inc $i}} {{ if gt $i 1}} OR  {{ end }}
			has(['{{ StringsJoin $val "','" }}'], labels.value[indexOf(labels.key, '{{ $key }}')])
		{{ end }}
)
{{ end }}
`

type Total struct {
	NumQueries    uint64  `db:"num_queries"`
	MQueryTimeCtn float32 `db:"m_query_time_cnt"`
	MQueryTimeSum float32 `db:"m_query_time_sum"`
	MQueryTimeMin float32 `db:"m_query_time_min"`
	MQueryTimeMax float32 `db:"m_query_time_max"`
	MQueryTimeP99 float32 `db:"m_query_time_p99"`
}

func (r *Reporter) GetTotal(from, to string, dbServers, dbSchemas, dbUsernames, clientHosts []string, dbLabels map[string][]string) (*Total, error) {
	arg := map[string]interface{}{
		"from":    from,
		"to":      to,
		"servers": dbServers,
		"schemas": dbSchemas,
		"users":   dbUsernames,
		"hosts":   clientHosts,
		"labels":  dbLabels,
	}
	var queryBuffer bytes.Buffer
	if tmpl, err := template.New("queryTotal").Funcs(funcMap).Parse(queryReportTotalTmpl); err != nil {
		log.Fatalln(err)
	} else if err = tmpl.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	res := Total{}
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	query, args, err = sqlx.In(query, args...)
	query = r.db.Rebind(query)
	err = r.db.Get(&res, query, args...)
	fmt.Printf("Total res: %v, %v \n", res, err)
	return &res, err
}

const queryReportTmpl = `
SELECT
digest AS digest1,
any(digest_text) AS digest_text1,

MIN(period_start) AS first_seen,
MAX(period_start) AS last_seen,

SUM(num_queries) AS num_queries,

SUM(m_query_time_cnt) AS m_query_time_cnt,
SUM(m_query_time_sum) AS m_query_time_sum,
MIN(m_query_time_min) AS m_query_time_min,
MAX(m_query_time_max) AS m_query_time_max,
AVG(m_query_time_p99) AS m_query_time_p99

FROM queries
WHERE period_start > :from AND period_start < :to
{{ if index . "first_seen" }} AND first_seen >= :from {{ end }}
{{ if index . "keyword" }} AND (digest = :keyword OR digest_text LIKE :start_keyword ) {{ end }}
{{ if index . "servers" }} AND db_server IN ( :servers ) {{ end }}
{{ if index . "schemas" }} AND db_schema IN ( :schemas ) {{ end }}
{{ if index . "users" }} AND db_username IN ( :users ) {{ end }}
{{ if index . "hosts" }} AND client_host IN ( :hosts ) {{ end }}
{{ if index . "labels" }}
	AND (
		{{$i := 0}}
		{{range $key, $val := index . "labels"}}
			{{ $i = inc $i}} {{ if gt $i 1}} OR {{ end }}
			has(['{{ StringsJoin $val "','" }}'], labels.value[indexOf(labels.key, '{{ $key }}')])
		{{ end }}
	)
{{ end }}
GROUP BY digest
ORDER BY ( :order ) DESC
LIMIT :offset, :limit
`

type QueryClassReport struct {
	Digest1       string  `db:"digest1"`
	DigestText1   string  `db:"digest_text1"`
	FirstSeen     string  `db:"first_seen"`
	LastSeen      string  `db:"last_seen"`
	NumQueries    uint64  `db:"num_queries"`
	MQueryTimeCtn float32 `db:"m_query_time_cnt"`
	MQueryTimeSum float32 `db:"m_query_time_sum"`
	MQueryTimeMin float32 `db:"m_query_time_min"`
	MQueryTimeMax float32 `db:"m_query_time_max"`
	MQueryTimeP99 float32 `db:"m_query_time_p99"`
}

func (r *Reporter) Select(from, to, keyword string, firstSeen bool, dbServers, dbSchemas, dbUsernames, clientHosts []string, dbLabels map[string][]string, order string, offset uint32, limit uint32) ([]*QueryClassReport, error) {

	fmt.Printf("HHHHH: %v, %v, %v \n", order, offset, limit)
	if order == "" {
		order = "m_query_time_sum"
	}

	if limit == 0 {
		limit = 10
	}
	arg := map[string]interface{}{
		"from":          from,
		"to":            to,
		"keyword":       keyword,
		"start_keyword": "%" + keyword,
		"first_seen":    firstSeen,
		"servers":       dbServers,
		"schemas":       dbSchemas,
		"users":         dbUsernames,
		"hosts":         clientHosts,
		"labels":        dbLabels,
		"order":         order,
		"offset":        offset,
		"limit":         limit,
	}
	var queryBuffer bytes.Buffer
	if tmpl, err := template.New("queryReport").Funcs(funcMap).Parse(queryReportTmpl); err != nil {
		log.Fatalln(err)
	} else if err = tmpl.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	res := []*QueryClassReport{}
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	query, args, err = sqlx.In(query, args...)
	query = r.db.Rebind(query)
	err = r.db.Select(&res, query, args...)
	fmt.Printf("Queries Classes res: %v, %v \n", res, err)
	return res, err
}

const serverReportTmpl = `
SELECT
db_server
groupUniqArray(db_schema) AS db_schemas,
groupUniqArray(db_username) AS db_usernames,
groupUniqArray(client_host) AS client_hosts,

MIN(period_start) AS first_seen,
MAX(period_start) AS last_seen,

SUM(num_queries) AS num_queries,

SUM(m_query_time_cnt) AS m_query_time_cnt,
SUM(m_query_time_sum) AS m_query_time_sum,
MIN(m_query_time_min) AS m_query_time_min,
MAX(m_query_time_max) AS m_query_time_max,
AVG(m_query_time_p99) AS m_query_time_p99

FROM queries
WHERE period_start > :from AND period_start < :to
{{ if index . "servers" }} AND db_server IN ( :servers ) {{ end }}
{{ if index . "schemas" }} AND db_schema IN ( :schemas ) {{ end }}
{{ if index . "users" }} AND db_username IN ( :users ) {{ end }}
{{ if index . "hosts" }} AND client_host IN ( :hosts ) {{ end }}
{{ if index . "labels" }}
	AND (
		{{$i := 0}}
		{{range $key, $val := index . "labels"}}
			{{ $i = inc $i}} {{ if gt $i 1}} OR  {{ end }}
			(label.key = ( :{{ $key }} ) AND labels.value IN ( :{{ $val }} ) )
		{{ end }}
	)
{{ end }}
GROUP BY db_server
ORDER BY m_query_time_sum DESC
`

const schemaReportTmpl = `
SELECT
db_schema
groupUniqArray(db_server) AS db_servers,
groupUniqArray(db_username) AS db_usernames,
groupUniqArray(client_host) AS client_hosts,

MIN(period_start) AS first_seen,
MAX(period_start) AS last_seen,

SUM(num_queries) AS num_queries,

SUM(m_query_time_cnt) AS m_query_time_cnt,
SUM(m_query_time_sum) AS m_query_time_sum,
MIN(m_query_time_min) AS m_query_time_min,
MAX(m_query_time_max) AS m_query_time_max,
AVG(m_query_time_p99) AS m_query_time_p99

FROM queries
WHERE period_start > :from AND period_start < :to
{{ if index . "servers" }} AND db_server IN ( :servers ) {{ end }}
{{ if index . "schemas" }} AND db_schema IN ( :schemas ) {{ end }}
{{ if index . "users" }} AND db_username IN ( :users ) {{ end }}
{{ if index . "hosts" }} AND client_host IN ( :hosts ) {{ end }}
{{ if index . "labels" }}
	AND (
		{{$i := 0}}
		{{range $key, $val := index . "labels"}}
			{{ $i = inc $i}} {{ if gt $i 1}} OR  {{ end }}
			(label.key = ( :{{ $key }} ) AND labels.value IN ( :{{ $val }} ) )
		{{ end }}
	)
{{ end }}
GROUP BY db_schema
ORDER BY m_query_time_sum DESC
`
