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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	Alias   string
	Derived bool
}

// planNode is a JSON value that keeps the order of an object's keys.
type planNode struct {
	keys     []string
	children []*planNode
	isObject bool
	isString bool
	str      string
}

func (n *planNode) get(key string) (*planNode, bool) {
	i := slices.Index(n.keys, key)
	if i < 0 {
		return nil, false
	}
	return n.children[i], true
}

func (n *planNode) text(key string) string {
	c, ok := n.get(key)
	if !ok || !c.isString {
		return ""
	}
	return c.str
}

func parsePlanNode(dec *json.Decoder) (*planNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		n := &planNode{isObject: v == '{'}
		for dec.More() {
			if n.isObject {
				key, err := dec.Token()
				if err != nil {
					return nil, err
				}
				n.keys = append(n.keys, fmt.Sprint(key))
			}
			child, err := parsePlanNode(dec)
			if err != nil {
				return nil, err
			}
			n.children = append(n.children, child)
		}
		_, err = dec.Token()
		if err != nil {
			return nil, err
		}
		return n, nil
	case string:
		return &planNode{isString: true, str: v}, nil
	default:
		return &planNode{}, nil
	}
}

// agentPlanKeys are the keys pmm-agent adds to the plan; they are not plan structure.
var agentPlanKeys = []string{"warnings", "real_table_name"}

// planTables lists the table entries of a MySQL JSON plan, format 1 or 2, in plan order.
func planTables(plan []byte) ([]planTable, error) {
	dec := json.NewDecoder(bytes.NewReader(plan))
	root, err := parsePlanNode(dec)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the plan: %w", err)
	}
	if !root.isObject {
		return nil, errors.New("the plan is not a JSON object")
	}
	var out []planTable
	collectPlanTables(root, &out)
	return out, nil
}

func collectPlanTables(n *planNode, out *[]planTable) {
	if n.isObject {
		if name := n.text("table_name"); name != "" {
			t := planTable{Schema: n.text("schema_name"), Name: name, Alias: n.text("alias")}
			_, materialized := n.get("materialized_from_subquery")
			_, hasInputs := n.get("inputs")
			t.Derived = materialized || (t.Schema == "" && hasInputs)
			if !slices.Contains(*out, t) {
				*out = append(*out, t)
			}
		}
	}
	for i, c := range n.children {
		if n.isObject && slices.Contains(agentPlanKeys, n.keys[i]) {
			continue
		}
		collectPlanTables(c, out)
	}
}

// aliasMap maps the alias, or the table name when there is none, of each FROM
// entry in Note 1003 to its table. An alias or name that stands for two different
// tables in the note is left out: it cannot be told apart.
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
	marks := sqlMarks(s, true)
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

// noteTables returns the FROM entries of a rewritten statement by alias or name.
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
	for i := range toks {
		if !toks[i].isWord("from") && !toks[i].isWord("join") {
			continue
		}
		j := i + 1
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

// resolveTables turns plan entries into tables, and names that cannot be turned
// into one are returned unresolved. Only names from tables are returned.
func resolveTables(tables []planTable, aliases map[string]tableRef, database string) ([]tableRef, []string) {
	var resolved []tableRef
	var unresolved []string
	for _, t := range tables {
		var ref tableRef
		ok := false
		switch {
		case t.Derived || strings.HasPrefix(t.Name, "<"):
		case t.Schema != "":
			ref, ok = tableRef{database: t.Schema, name: t.Name}, true
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

// decodeExplainEnvelope returns the plan inside the explain envelope
// {explain_result: <base64 plan>, explained_query, is_dml}. The explained
// statement is not returned.
func decodeExplainEnvelope(output string) ([]byte, error) {
	var envelope struct {
		ExplainResult string `json:"explain_result"`
	}
	err := json.Unmarshal([]byte(output), &envelope)
	if err != nil {
		return nil, fmt.Errorf("explain output is not an envelope: %w", err)
	}
	if envelope.ExplainResult == "" {
		return nil, errors.New("explain output has no explain_result")
	}
	plan, err := base64.StdEncoding.DecodeString(envelope.ExplainResult)
	if err != nil {
		return nil, fmt.Errorf("explain_result is not base64: %w", err)
	}
	return plan, nil
}
