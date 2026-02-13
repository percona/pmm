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

package parser

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var cursorOperation = []string{
	"sort", "limit", "skip", "batchSize", "explain",
	"forEach", "map", "max", "min", "pretty", "size", "toArray", "tryNext",
}

// parseCursorOperations parses query cursor operations from raw bson document returned by currentOp command into string.
// Stages are places like:
// db.collection.find(...).operation1(...).operation2(...)
// Example: db.collection.find(...).sort(...).limit(5).skip(10).batchSize(1)
func parseCursorOperations(raw bson.Raw) string {
	var parsedOperations []string

	for _, key := range cursorOperation {
		if phaseRaw := raw.Lookup(key); !phaseRaw.IsZero() {
			parsedOperations = append(parsedOperations,
				fmt.Sprintf("%s(%s)", key, parseRawValue(phaseRaw)))
		}
	}
	return strings.Join(parsedOperations, ".")
}
