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

// Package queryparser provides functionality for queries parsing.
package queryparser

import (
	"regexp"
	"strings"
)

var (
	mySQLQuotedRegexp  = regexp.MustCompile(`'([^'\\]|\\.)*'|"([^"\\]|\\.)*"`)
	mySQLCommentRegexp = regexp.MustCompile(`(?m)--.*$|#.*$|(?s)/\*.*?\*/`)
	pgQuotedRegexp     = regexp.MustCompile(`'([^'\\]|\\.)*'|"([^"\\]|\\.)*"|\$\$.*?\$\$`)
	pgCommentRegexp    = regexp.MustCompile(`(?m)--.*$|(?s)/\*.*?\*/`)
	keyValueRegexp     = regexp.MustCompile(`(?s)([^'",\s]+)\s*=\s*'([^']*?)'`)
)

func parseMySQLComments(query string) (map[string]string, error) {
	return parseComments(query, mySQLQuotedRegexp, mySQLCommentRegexp)
}

func parsePGComments(query string) (map[string]string, error) {
	return parseComments(query, pgQuotedRegexp, pgCommentRegexp)
}

func parseComments(query string, quotedRegexp *regexp.Regexp, commentRegexp *regexp.Regexp) (map[string]string, error) {
	result := make(map[string]string)
	comments := extractComments(query, quotedRegexp, commentRegexp)
	for _, c := range comments {
		for k, v := range parseKeyValueFromComment(c) {
			result[k] = v
		}
	}

	return result, nil
}

func extractComments(query string, quotedRegexp, commentRegexp *regexp.Regexp) []string {
	// Find quoted spans to ignore comments inside
	quotedSpans := quotedRegexp.FindAllStringIndex(query, -1)

	isInsideQuotes := func(pos int) bool {
		for _, span := range quotedSpans {
			if pos >= span[0] && pos < span[1] {
				return true
			}
		}
		return false
	}

	matches := commentRegexp.FindAllStringIndex(query, -1)
	comments := []string{}
	for _, m := range matches {
		if !isInsideQuotes(m[0]) {
			comment := strings.TrimSpace(query[m[0]:m[1]])
			comments = append(comments, comment)
		}
	}
	return comments
}

func parseKeyValueFromComment(s string) map[string]string {
	res := make(map[string]string)
	matches := keyValueRegexp.FindAllStringSubmatch(s, -1)
	for _, v := range matches {
		if len(v) < 3 {
			continue
		}
		res[v[1]] = v[2]
	}

	return res
}
