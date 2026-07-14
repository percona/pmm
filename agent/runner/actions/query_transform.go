// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actions

import (
	"fmt"
	"regexp"
	"strings"
)

//nolint:lll
var (
	dmlVerbs         = []string{"select", "insert", "update", "delete", "replace"}
	commentsRe       = regexp.MustCompile(`(?s)\/\*(.*?)\*\/`)
	selectRe         = regexp.MustCompile(`(?i)^select\s+(.*?)\bfrom\s+(.*?)$`)
	updateRe         = regexp.MustCompile(`(?i)^update\s+(?:low_priority|ignore)?\s*(.*?)\s+set\s+(.*?)(?:\s+where\s+(.*?))?(?:\s+limit\s*[0-9]+(?:\s*,\s*[0-9]+)?)?$`)
	deleteRe         = regexp.MustCompile(`(?i)^delete\s+(.*?)\bfrom\s+(.*?)$`)
	insertRe         = regexp.MustCompile(`(?i)^(?:insert(?:\s+ignore)?|replace)\s+.*?\binto\s+(.*?)\(([^\)]+)\)\s*values?\s*\((.*?)\)\s*(?:\slimit\s|on\s+duplicate\s+key.*)?\s*$`)
	insertReNoFields = regexp.MustCompile(`(?i)^(?:insert(?:\s+ignore)?|replace)\s+.*?\binto\s+(.*?)\s*values?\s*\((.*?)\)\s*(?:\slimit\s|on\s+duplicate\s+key.*)?\s*$`)
	insertSetRe      = regexp.MustCompile(`(?i)(?:insert(?:\s+ignore)?|replace)\s+(?:.*?\binto)\s+(.*?)\s*set\s+(.*?)\s*(?:\blimit\b|on\s+duplicate\s+key.*)?\s*$`)
)

func prepareQuery(query string) string {
	query = commentsRe.ReplaceAllString(query, "")
	query = strings.ReplaceAll(query, "\t", " ")
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.TrimRight(query, ";")
	return strings.TrimLeft(query, " ")
}

func isDMLQuery(query string) bool {
	query = strings.ToLower(prepareQuery(query))
	for _, verb := range dmlVerbs {
		if strings.HasPrefix(query, verb) {
			return true
		}
	}
	return false
}

/*
MySQL version prior 5.6.3 cannot run explain on DML commands.
Newer MySQL versions can run explain on DML queries but higher persmissions
are needed and the pmm user is a not privileged user.
This function converts DML queries to the equivalent SELECT to make
it able to explain DML queries on older MySQL versions and for unprivileged users.
*/

// dmlToSelect returns query converted to select and boolean, if conversion were needed.
func dmlToSelect(query string) (string, bool) {
	query = prepareQuery(query)

	m := selectRe.FindStringSubmatch(query)
	if len(m) > 1 {
		return query, false
	}

	m = updateRe.FindStringSubmatch(query)
	// > 2 because we need at least a table name and a list of fields
	if len(m) > 2 {
		return updateToSelect(m), true
	}

	m = deleteRe.FindStringSubmatch(query)
	if len(m) > 1 {
		return deleteToSelect(m), true
	}

	m = insertRe.FindStringSubmatch(query)
	if len(m) > 2 {
		return insertToSelect(m), true
	}

	m = insertSetRe.FindStringSubmatch(query)
	if len(m) > 2 {
		return insertWithSetToSelect(m), true
	}

	m = insertReNoFields.FindStringSubmatch(query)
	if len(m) > 2 {
		return insertToSelectNoFields(m), true
	}

	return "", false
}

func updateToSelect(matches []string) string {
	matches = matches[1:]
	matches[0], matches[1] = matches[1], matches[0]
	format := []string{"SELECT %s", " FROM %s", " WHERE %s"}
	result := ""
	for i, match := range matches {
		if match != "" {
			result += fmt.Sprintf(format[i], match)
		}
	}
	return result
}

func deleteToSelect(matches []string) string {
	if strings.Contains(matches[2], "join") {
		return fmt.Sprintf("SELECT 1 FROM %s", matches[2])
	}
	return fmt.Sprintf("SELECT * FROM %s", matches[2])
}

func insertToSelect(matches []string) string {
	fields := strings.Split(matches[2], ",")
	values := strings.Split(matches[3], ",")
	if len(fields) == len(values) {
		query := fmt.Sprintf("SELECT * FROM %s WHERE ", matches[1])
		sep := ""
		for i := 0; i < len(fields); i++ {
			query += fmt.Sprintf(`%s%s=%s`, sep, strings.TrimSpace(fields[i]), values[i])
			sep = " and "
		}
		return query
	}
	return fmt.Sprintf("SELECT * FROM %s LIMIT 1", matches[1])
}

func insertToSelectNoFields(matches []string) string {
	return fmt.Sprintf("SELECT * FROM %s LIMIT 1", matches[1])
}

func insertWithSetToSelect(matches []string) string {
	return fmt.Sprintf("SELECT * FROM %s WHERE %s", matches[1], strings.ReplaceAll(matches[2], ",", " AND "))
}
