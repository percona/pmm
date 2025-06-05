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

// Package truncate privides strings truncation utilities.
package truncate

import "strings"

const (
	defaultMaxQueryLength        = int32(2048)
	defaultMongoDBMaxQueryLength = int32(4096)
)

// Query truncate query to specific length of chars, if needed. -1: No limit, 0: Default (2048, Mongo 4096).
// Also truncate all invalid UTF-8 chars.
func Query(q string, maxQueryLength, defaultMaxQueryLength int32) (string, bool) {
	if maxQueryLength < 0 {
		return string([]rune(q)), false
	}

	if maxQueryLength == 0 {
		maxQueryLength = defaultMaxQueryLength
	}

	runes := []rune(q)
	if int32(len(runes)) <= maxQueryLength {
		return string(runes), strings.HasSuffix(string(runes), "...") || len(runes) != len(q)
	}

	// for queries shorter than 4 chars
	if maxQueryLength < 4 {
		return " ...", true
	}

	// copy MySQL behavior
	return string(runes[:maxQueryLength-4]) + " ...", true
}

// GetDefaultMaxQueryLength returns default decimal value for query length.
func GetDefaultMaxQueryLength() int32 {
	return defaultMaxQueryLength
}

// GetMongoDBDefaultMaxQueryLength returns default decimal value for MongoDB query length.
func GetMongoDBDefaultMaxQueryLength() int32 {
	return defaultMongoDBMaxQueryLength
}
