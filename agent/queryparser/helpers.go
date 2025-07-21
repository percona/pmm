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
	"sync"
)

var (
	mySQLQuotedRegexp *regexp.Regexp
	mySQLQuotedOnce   sync.Once
	errMySQLQuoted    error

	mySQLCommentRegexp *regexp.Regexp
	mySQLCommentOnce   sync.Once
	errMySQLComment    error

	pgQuotedRegexp *regexp.Regexp
	pgQuotedOnce   sync.Once
	errPGQuoted    error

	pgCommentRegexp *regexp.Regexp
	pgCommentOnce   sync.Once
	errPGComment    error

	keyValueRegexp *regexp.Regexp
	keyValueOnce   sync.Once
	errKeyValue    error
)

func parseMySQLComments(query string) (map[string]bool, error) {
	if err := prepareMySQLComments(); err != nil {
		return nil, err
	}

	return parseComments(query, mySQLQuotedRegexp, mySQLCommentRegexp)
}

func parsePGComments(query string) (map[string]bool, error) {
	if err := preparePGComments(); err != nil {
		return nil, err
	}

	return parseComments(query, pgQuotedRegexp, pgCommentRegexp)
}

func parseComments(query string, quotedRegexp *regexp.Regexp, commentRegexp *regexp.Regexp) (map[string]bool, error) {
	result := make(map[string]bool)
	comments := extractComments(query, quotedRegexp, commentRegexp)
	for _, v := range comments {
		parsed, err := parseKeyValueFromComment(string(v))
		if err != nil {
			continue
		}
		for k := range parsed {
			result[k] = true
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
			// Clean comment: trim trailing spaces and newlines
			comment := strings.TrimSpace(query[m[0]:m[1]])
			comments = append(comments, comment)
		}
	}
	return comments
}

func parseKeyValueFromComment(s string) (map[string]bool, error) {
	if err := prepareKeyValueRegexp(); err != nil {
		return nil, err
	}

	res := make(map[string]bool)
	matches := keyValueRegexp.FindAllStringSubmatch(s, -1)
	for _, v := range matches {
		if len(v) < 2 {
			continue
		}
		res[v[1]] = true
	}

	return res, nil
}

func prepareMySQLComments() error {
	// to compile regexp only once
	mySQLQuotedOnce.Do(func() {
		mySQLQuotedRegexp, errMySQLQuoted = regexp.Compile(`'([^'\\]|\\.)*'|"([^"\\]|\\.)*"`) //nolint:gocritic
	})
	if errMySQLQuoted != nil {
		return errMySQLQuoted
	}

	mySQLCommentOnce.Do(func() {
		mySQLCommentRegexp, errMySQLComment = regexp.Compile(`(?m)--.*$|#.*$|(?s)/\*.*?\*/`) //nolint:gocritic
	})
	if errMySQLComment != nil {
		return errMySQLComment
	}

	return nil
}

func preparePGComments() error {
	// to compile regexp only once
	pgQuotedOnce.Do(func() {
		pgQuotedRegexp, errPGQuoted = regexp.Compile(`'([^'\\]|\\.)*'|"([^"\\]|\\.)*"|\$\$.*?\$\$`) //nolint:gocritic
	})
	if errPGQuoted != nil {
		return errPGQuoted
	}

	pgCommentOnce.Do(func() {
		pgCommentRegexp, errPGComment = regexp.Compile(`(?m)--.*$|(?s)/\*.*?\*/`) //nolint:gocritic
	})
	if errPGComment != nil {
		return errPGComment
	}

	return nil
}

func prepareKeyValueRegexp() error {
	// to compile regexp only once
	keyValueOnce.Do(func() {
		keyValueRegexp, errKeyValue = regexp.Compile(`(?s)([a-zA-Z-\d]+='.+?')`) //nolint:gocritic
	})
	if errKeyValue != nil {
		return errKeyValue
	}

	return nil
}
