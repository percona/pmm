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

// Package mysql contains shared and helpers functions for mysql agent.
package mysql

import (
	"fmt"
	"strings"
)

// QueryIDWithSchema returns query ID with schema in format schema-queryID.
// It is used to fix: https://perconadev.atlassian.net/browse/PMM-12413.
func QueryIDWithSchema(schema, queryID string) string {
	if schema == "" {
		return queryID
	}

	return fmt.Sprintf("%s-%s", schema, queryID)
}

// QueryIDWithSchema returns plan query ID without schema.
// It is used to fix: https://perconadev.atlassian.net/browse/PMM-12413.
func QueryIDWithoutSchema(queryID string) string {
	res := strings.Split(queryID, "-")
	if len(res) < 2 {
		return queryID
	}

	return res[1]
}
