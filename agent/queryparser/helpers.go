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
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	spaceRegexp *regexp.Regexp
	spaceOnce   sync.Once
	errSpace    error

	multilineRegexp *regexp.Regexp
	multilineOnce   sync.Once
	errMultiline    error

	dashRegexp *regexp.Regexp
	dashOnce   sync.Once
	errDash    error

	hashRegexp *regexp.Regexp
	hashOnce   sync.Once
	errHash    error

	keyValueRegexp *regexp.Regexp
	keyValueOnce   sync.Once
	errKeyValue    error
)

func parseMySQLComments(q string) (map[string]bool, error) {
	if err := prepareMultilineRegexp(); err != nil {
		return nil, err
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

		parsed, err := parseKeyValueFromComment(value)
		if err != nil {
			continue
		}
		for k := range parsed {
			comments[k] = true
		}
	}

	hashComments, err := parseSinglelineComments(q, "#")
	if err != nil {
		return nil, err
	}
	for c := range hashComments {
		parsed, err := parseKeyValueFromComment(c)
		if err != nil {
			continue
		}
		for k := range parsed {
			comments[k] = true
		}
	}

	dashComments, err := parseSinglelineComments(q, "--")
	if err != nil {
		return nil, err
	}
	for c := range dashComments {
		parsed, err := parseKeyValueFromComment(c)
		if err != nil {
			continue
		}
		for k := range parsed {
			comments[k] = true
		}
	}

	return comments, nil
}

func parsePostgreSQLComments(q string) (map[string]bool, error) {
	if err := prepareMultilineRegexp(); err != nil {
		return nil, err
	}

	// comments using comment as a key to avoid duplicates
	comments := make(map[string]bool)
	for _, v := range multilineRegexp.FindAllStringSubmatch(q, -1) {
		if len(v) < 2 {
			continue
		}

		value := removeFormatting(v[1])

		var err error
		value, err = removeSpaces(strings.ReplaceAll(value, "*", ""))
		if err != nil {
			return nil, err
		}

		parsed, err := parseKeyValueFromComment(value)
		if err != nil {
			continue
		}
		for k := range parsed {
			comments[k] = true
		}
	}

	dashComments, err := parseSinglelineComments(q, "--")
	if err != nil {
		return nil, err
	}
	for c := range dashComments {
		parsed, err := parseKeyValueFromComment(c)
		if err != nil {
			continue
		}
		for k := range parsed {
			comments[k] = true
		}
	}

	return comments, nil
}

func parseSinglelineComments(q, startChar string) (map[string]bool, error) {
	var r *regexp.Regexp
	switch startChar {
	case "--":
		if err := prepareDashRegexp(); err != nil {
			return nil, err
		}
		r = dashRegexp
	case "#":
		if err := prepareHashRegexp(); err != nil {
			return nil, err
		}
		r = hashRegexp
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

func parseKeyValueFromComment(s string) (map[string]bool, error) {
	if err := prepareKeyValueRegexp(); err != nil {
		return nil, err
	}

	res := make(map[string]bool)
	matches := keyValueRegexp.FindAllStringSubmatch(removeFormatting(s), -1)
	for _, v := range matches {
		if len(v) < 2 {
			continue
		}
		res[v[1]] = true
	}

	return res, nil
}

func prepareMultilineRegexp() error {
	// to compile regexp only once
	multilineOnce.Do(func() {
		multilineRegexp, errMultiline = regexp.Compile(`(?s)\/\*(.*?)\*\/`) //nolint:gocritic
	})
	if errMultiline != nil {
		return errMultiline
	}

	return nil
}

func prepareSpaceRegexp() error {
	// to compile regexp only once
	spaceOnce.Do(func() {
		spaceRegexp, errSpace = regexp.Compile(`\s+`) //nolint:gocritic
	})
	if errSpace != nil {
		return errSpace
	}

	return nil
}

func prepareDashRegexp() error {
	// to compile regexp only once
	dashOnce.Do(func() {
		dashRegexp, errDash = regexp.Compile(`--.*`) //nolint:gocritic
	})
	if errDash != nil {
		return errDash
	}

	return nil
}

func prepareHashRegexp() error {
	// to compile regexp only once
	hashOnce.Do(func() {
		hashRegexp, errHash = regexp.Compile(`#.*`) //nolint:gocritic
	})
	if errHash != nil {
		return errHash
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

func removeFormatting(s string) string {
	value := strings.ReplaceAll(s, "\n", "")
	return strings.ReplaceAll(value, "\t", "")
}

func removeSpaces(s string) (string, error) {
	if err := prepareSpaceRegexp(); err != nil {
		return "", err
	}

	value := spaceRegexp.ReplaceAllString(s, " ")
	value = strings.TrimLeft(value, " ")
	return strings.TrimRight(value, " "), nil
}

func stringToLines(s string) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err := scanner.Err()

	return lines, err
}
