// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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

	"go.mongodb.org/mongo-driver/bson"
)

// https://mongodb.github.io/node-mongodb-native/Next/interfaces/AggregateOptions.html
var aggregateOptions = []string{
	"allowDiskUse", "authdb", "batchSize",
	"bsonRegExp", "bypassDocumentValidation", "checkKeys", "collation", "comment",
	"cursor", "dbName", "enableUtf8Validation", "explain", "fieldsAsRaw", "hint", "let",
	"ignoreUndefined", "maxAwaitTimeMS", "maxTimeMS", "out", "promoteBuffers",
	"promoteLongs", "promoteValues", "raw", "readConcern", "readPreference",
	"serializeFunctions", "session", "timeoutMS", "useBigInt64", "willRetryWrite",
	"writeConcern",
}

func parseCommandAggregate(commandRaw bson.Raw, ns string) string {
	// command has format:
	// https://www.mongodb.com/docs/manual/reference/method/db.collection.aggregate/#syntax
	// db.collection.aggregate(<pipeline>, <options>).<cursorOperations>
	aggPlaceholders := []string{parseDocument(commandRaw, "pipeline")}
	if options := parseOptions(commandRaw, aggregateOptions); options != "" {
		aggPlaceholders = append(aggPlaceholders, options)
	}

	queryPlaceholders := []string{
		fmt.Sprintf(`%s(%s)`, ns, strings.Join(aggPlaceholders, ", ")),
	}

	if cursorOperations := parseCursorOperations(commandRaw); cursorOperations != "" {
		queryPlaceholders = append(queryPlaceholders, cursorOperations)
	}

	return strings.Join(queryPlaceholders, ".")
}
