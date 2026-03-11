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

// https://www.mongodb.com/docs/manual/reference/method/db.collection.deleteMany/#syntax
var deleteOptions = []string{"writeConcern", "collation", "hint", "let", "maxTimeMS"}

func parseCommandDelete(commandRaw bson.Raw, collectionName string) string {
	// command has format:
	// https://www.mongodb.com/docs/manual/reference/method/db.collection.deleteMany/#syntax
	// db.<collection>.<deleteOne|deleteMany>(<filter>, <options>)
	deletePlaceholders := []string{
		parseDocument(commandRaw, "q"),
	}
	if params := parseOptions(commandRaw, deleteOptions); params != "" {
		deletePlaceholders = append(deletePlaceholders, params)
	}

	method := "deleteMany"
	// If limit is 1, then it's deleteOne, otherwise it's deleteMany.
	// By default, if limit is not specified, it's deleteMany.
	if limit, ok := commandRaw.Lookup("limit").Int32OK(); ok && limit == 1 {
		method = "deleteOne"
	}

	queryPlaceholders := []string{
		"db",
		collectionName,
		fmt.Sprintf("%s(%s)", method, strings.Join(deletePlaceholders, ", ")),
	}

	return strings.Join(queryPlaceholders, ".")
}
