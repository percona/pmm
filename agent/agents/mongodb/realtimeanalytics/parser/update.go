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
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// https://www.mongodb.com/docs/manual/reference/method/db.collection.update/#syntax
var updateOptions = []string{
	"multi", "upsert", "writeConcern", "collation",
	"arrayFilters", "hint", "let", "maxTimeMS", "bypassDocumentValidation",
}

func parseCommandUpdate(commandRaw bson.Raw, collectionName string) string {
	// command has format:
	// db.<collection>.update(<query>, <update>, <options>)
	// https://www.mongodb.com/docs/manual/reference/method/db.collection.update/#syntax
	updatePlaceholders := []string{
		parseDocument(commandRaw, "q"),
		parseDocument(commandRaw, "u"),
	}
	if options := parseOptions(commandRaw, updateOptions); options != "" {
		updatePlaceholders = append(updatePlaceholders, options)
	}

	updatePlaceholders = slices.DeleteFunc(updatePlaceholders, func(s string) bool {
		return s == ""
	})

	queryPlaceholders := []string{
		"db",
		collectionName,
		fmt.Sprintf("update(%s)", strings.Join(updatePlaceholders, ", ")),
	}

	return strings.Join(queryPlaceholders, ".")
}
