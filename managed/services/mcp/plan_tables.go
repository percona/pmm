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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

// This file finds the tables a MySQL statement touches from its EXPLAIN FORMAT=JSON
// plan, so a schema can be fetched from a queryid alone. Format 1 names a table by
// its alias and carries no schema, so the real names come from Note 1003 (the
// rewritten statement SHOW WARNINGS returns after the EXPLAIN); format 2 carries
// the real name, schema and alias on every table.

// planTable is one table entry of a plan.
type planTable struct {
	Schema  string
	Name    string
	Derived bool
}

// decodeMySQLPlan decodes a MySQL JSON plan, its numbers as json.Number. As pmm-agent decodes MySQL's plan into a map
// and encodes it again, every object's keys arrive sorted, and a map loses no order of theirs.
func decodeMySQLPlan(plan []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(plan))
	dec.UseNumber()
	var root any
	err := dec.Decode(&root)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the plan: %w", err)
	}
	object, ok := root.(map[string]any)
	if !ok {
		return nil, errors.New("the plan is not a JSON object")
	}
	_, err = dec.Token()
	if !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after JSON value")
	}
	return object, nil
}

// planTables lists the table entries of a MySQL JSON plan, format 1 or 2, in plan order, an object's keys taken in
// sorted order.
func planTables(plan []byte) ([]planTable, error) {
	root, err := decodeMySQLPlan(plan)
	if err != nil {
		return nil, err
	}
	var out []planTable
	collectPlanTables(root, &out)
	return out, nil
}

func collectPlanTables(v any, out *[]planTable) {
	switch v := v.(type) {
	case map[string]any:
		name, _ := v["table_name"].(string)
		if name != "" {
			schema, _ := v["schema_name"].(string)
			_, materialized := v["materialized_from_subquery"]
			_, hasInputs := v["inputs"]
			*out = append(*out, planTable{Schema: schema, Name: name, Derived: materialized || (schema == "" && hasInputs)})
		}
		for _, key := range slices.Sorted(maps.Keys(v)) {
			collectPlanTables(v[key], out)
		}
	case []any:
		for _, c := range v {
			collectPlanTables(c, out)
		}
	}
}

// aliasMap maps the alias, or the table name when there is none, of each table
// noteTables finds in Note 1003 to its table. An alias or name that stands for two
// different tables in the note is left out: it cannot be told apart.
func aliasMap(plan []byte) map[string]tableRef {
	var envelope struct {
		Warnings []struct {
			Code    any    `json:"Code"`
			Message string `json:"Message"`
		} `json:"warnings"`
	}
	err := json.Unmarshal(plan, &envelope)
	if err != nil {
		return map[string]tableRef{}
	}
	for _, w := range envelope.Warnings {
		if fmt.Sprint(w.Code) == "1003" {
			return noteTables(w.Message)
		}
	}
	return map[string]tableRef{}
}

// noteToken is a word, a quoted identifier or a punctuation mark of a note.
type noteToken struct {
	text   string
	quoted bool
	punct  bool
}

func (t noteToken) isWord(w string) bool {
	return !t.quoted && !t.punct && strings.EqualFold(t.text, w)
}

func (t noteToken) isIdent() bool {
	return t.quoted || (!t.punct && !t.isWord("select"))
}

// tokenizeNote splits a rewritten statement into tokens, leaving out literals and
// comments.
func tokenizeNote(s string) []noteToken {
	marks := sqlMarks(s, true, false)
	var out []noteToken
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case marks[i] != 0 || c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '#':
			return nil
		case strings.HasPrefix(s[i:], "/*"), strings.HasPrefix(s[i:], "*/"), strings.HasPrefix(s[i:], "--"):
			i += 2
		case c == '`':
			var b strings.Builder
			i++
			for i < len(s) {
				if marks[i] != 0 {
					return nil
				}
				if s[i] == '`' {
					if i+1 < len(s) && s[i+1] == '`' {
						b.WriteByte('`')
						i += 2
						continue
					}
					break
				}
				b.WriteByte(s[i])
				i++
			}
			i++
			out = append(out, noteToken{text: b.String(), quoted: true})
		case isIdentByte(c):
			j := i
			for j < len(s) && isIdentByte(s[j]) && marks[j] == 0 {
				j++
			}
			out = append(out, noteToken{text: s[i:j]})
			i = j
		default:
			out = append(out, noteToken{text: string(c), punct: true})
			i++
		}
	}
	return out
}

// noteTables returns the tables of a rewritten statement by alias or name: the FROM entries of a SELECT or DELETE, the
// JOIN and STRAIGHT_JOIN entries, the table of an UPDATE, and the INTO table of an INSERT or REPLACE.
func noteTables(note string) map[string]tableRef {
	toks := tokenizeNote(note)

	ctes := map[string]bool{}
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].isIdent() && toks[i+1].isWord("as") && toks[i+2].text == "(" && toks[i+2].punct {
			ctes[toks[i].text] = true
		}
	}

	out := map[string]tableRef{}
	ambiguous := map[string]bool{}
	// queries holds, for the statement and each open parenthesis, whether a SELECT or DELETE started in it, so that the
	// FROM of a function such as EXTRACT(YEAR FROM ...) is not read as a clause.
	queries := []bool{false}
	for i := range toks {
		table := false
		switch {
		case toks[i].punct && toks[i].text == "(":
			queries = append(queries, false)
		case toks[i].punct && toks[i].text == ")" && len(queries) > 1:
			queries = queries[:len(queries)-1]
		case toks[i].isWord("select") || toks[i].isWord("delete"):
			queries[len(queries)-1] = true
		case toks[i].isWord("from"):
			table = queries[len(queries)-1]
		case toks[i].isWord("join") || toks[i].isWord("straight_join"):
			table = true
		// The UPDATE of FOR UPDATE and of ON DUPLICATE KEY UPDATE names no table.
		case toks[i].isWord("update"):
			table = i == 0 || !toks[i-1].isWord("for") && !toks[i-1].isWord("key")
		case toks[i].isWord("into"):
			table = toks[0].isWord("insert") || toks[0].isWord("replace")
		}
		if !table {
			continue
		}
		j := i + 1
		for toks[i].isWord("update") && j < len(toks) && (toks[j].isWord("low_priority") || toks[j].isWord("ignore")) {
			j++
		}
		if j+1 < len(toks) && toks[j].punct && toks[j].text == "(" && toks[j+1].isIdent() {
			j++
		}
		if j >= len(toks) || !toks[j].isIdent() {
			continue
		}
		ref := tableRef{name: toks[j].text}
		j++
		if j+1 < len(toks) && toks[j].punct && toks[j].text == "." && toks[j+1].isIdent() {
			ref.database, ref.name = ref.name, toks[j+1].text
			j += 2
		}
		if j < len(toks) && toks[j].punct && toks[j].text == "." {
			continue
		}
		if ref.database == "" && ctes[ref.name] {
			continue
		}
		j = skipPartitionList(toks, j)
		key := ref.name
		if j < len(toks) && toks[j].quoted {
			key = toks[j].text
		}
		prev, ok := out[key]
		if ok && prev != ref {
			ambiguous[key] = true
		}
		out[key] = ref
	}
	for key := range ambiguous {
		delete(out, key)
	}
	return out
}

// skipPartitionList returns the index past the PARTITION (`p0`,…) list that MySQL prints between a table and its
// alias when the statement selects partitions, or j when toks[j] starts none.
func skipPartitionList(toks []noteToken, j int) int {
	if j+1 >= len(toks) || !toks[j].isWord("partition") || !toks[j+1].punct || toks[j+1].text != "(" {
		return j
	}
	for j < len(toks) && (!toks[j].punct || toks[j].text != ")") {
		j++
	}
	return j + 1
}

// resolveTables turns plan entries into tables, and names that cannot be turned
// into one are returned unresolved. Only names from tables are returned.
func resolveTables(tables []planTable, aliases map[string]tableRef, database string) ([]tableRef, []string) {
	var resolved []tableRef
	var unresolved []string
	for _, t := range tables {
		var ref tableRef
		ok := false
		switch {
		case t.Derived:
		case t.Schema != "":
			ref, ok = tableRef{database: t.Schema, name: t.Name}, true
		// MySQL's own names, such as <union1,2>, carry no schema; a base table may be named <…> too.
		case strings.HasPrefix(t.Name, "<"):
		default:
			ref, ok = aliases[t.Name]
			if ok && ref.database == "" {
				ref.database = database
			}
		}
		switch {
		case !ok:
			if !slices.Contains(unresolved, t.Name) {
				unresolved = append(unresolved, t.Name)
			}
		case !slices.Contains(resolved, ref):
			resolved = append(resolved, ref)
		}
	}
	return resolved, unresolved
}
