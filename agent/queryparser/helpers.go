// Copyright 2019 Percona LLC
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
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	spaceRegexp *regexp.Regexp
	spaceError  error
	spaceOnce   sync.Once

	multilineRegexp *regexp.Regexp
	multilineError  error
	multilineOnce   sync.Once
)

func parseMySQLComments(q string) (map[string]bool, error) {
	prepareMultilineRegexp()
	if multilineError != nil {
		return nil, multilineError
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	for _, v := range multilineRegexp.FindAllStringSubmatch(q, -1) {
		if len(v) < 2 {
			continue
		}

		value := removeFormatting(v[1])

		var err error
		value, err = removeSpaces(value)
		if err != nil {
			return nil, err
		}

		comments[value] = true
	}

	hashComments, err := parseSinglelineComments(q, "#")
	if err != nil {
		return nil, err
	}
	for c := range hashComments {
		comments[c] = true
	}

	dashComments, err := parseSinglelineComments(q, "--")
	if err != nil {
		return nil, err
	}
	for c := range dashComments {
		comments[c] = true
	}

	return comments, nil
}

func parsePostgreSQLComments(q string) (map[string]bool, error) {
	prepareMultilineRegexp()
	if multilineError != nil {
		return nil, multilineError
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	for _, v := range multilineRegexp.FindAllStringSubmatch(q, -1) {
		if len(v) < 2 {
			continue
		}

		value := removeFormatting(v[1])

		var err error
		value, err = removeSpaces(value)
		if err != nil {
			return nil, err
		}

		comments[value] = true
	}

	dashComments, err := parseSinglelineComments(q, "--")
	if err != nil {
		return nil, err
	}
	for c := range dashComments {
		comments[c] = true
	}

	return comments, nil
}

func parseSinglelineComments(q, startChar string) (map[string]bool, error) {
	r, err := regexp.Compile(fmt.Sprintf("%s.*", startChar))
	if err != nil {
		return nil, err
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	lines, err := stringToLines(q)
	if err != nil {
		return nil, err
	}
	for _, l := range lines {
		for _, v := range r.FindStringSubmatch(l) {
			comments[strings.TrimLeft(v, fmt.Sprintf("%s ", startChar))] = true
		}
	}

	return comments, nil
}

func prepareMultilineRegexp() error {
	// To compile regexp only once.
	multilineOnce.Do(func() {
		multilineRegexp, multilineError = regexp.Compile(`(?s)\/\*(.*?)\*\/`)
	})
	if multilineError != nil {
		return multilineError
	}

	return nil
}

func removeFormatting(s string) string {
	value := strings.ReplaceAll(s, "\n", "")
	return strings.ReplaceAll(value, "\t", "")
}

func removeSpaces(s string) (string, error) {
	// To compile regexp only once.
	spaceOnce.Do(func() {
		spaceRegexp, spaceError = regexp.Compile(`\s+`)
	})
	if spaceError != nil {
		return "", spaceError
	}

	value := spaceRegexp.ReplaceAllString(s, " ")
	value = strings.TrimLeft(value, " ")
	return strings.TrimRight(value, " "), nil
}

func stringToLines(s string) (lines []string, err error) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()

	return
}
