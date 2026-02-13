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
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// https://www.mongodb.com/docs/manual/reference/method/db.collection.find/#options
var findOptions = []string{
	"allowDiskUse", "allowPartialResults", "awaitData", "collation", "comment",
	"explain", "hint", "max", "maxAwaitTimeMS", "maxTimeMS",
	"min", "noCursorTimeout", "readConcern", "readPreference",
	"returnKey", "showRecordId", "tailable", "limit", "skip", "sort",
}

func parseCommandFind(commandRaw bson.Raw) string {
	collectionName, _ := commandRaw.Lookup("find").StringValueOK()

	// query has format:
	// https://www.mongodb.com/docs/manual/reference/method/db.collection.find/#syntax
	// db.<collection>.find(<filter>, <projection>, <options>).<phases>
	filter := parseDocument(commandRaw, "filter")
	projection := parseDocument(commandRaw, "projection")
	options := parseOptions(commandRaw, findOptions)

	// possible cases:
	// db.<collection>.find()

	// db.<collection>.find(<filter>)
	// db.<collection>.find(<filter>, <projection>)
	// db.<collection>.find(<filter>, <projection>, <options>)
	// db.<collection>.find(<filter>, {}, <options>)

	// db.<collection>.find({}, <projection>)
	// db.<collection>.find({}, <projection>, <options>)

	// db.<collection>.find({}, {}, <options>)
	if options != "" {
		if projection == "" {
			projection = "{}"
		}

		if filter == "" {
			filter = "{}"
		}
	}

	if projection != "" {
		if filter == "" {
			filter = "{}"
		}
	}

	findPlaceholders := []string{filter, projection, options}
	findPlaceholders = slices.DeleteFunc(findPlaceholders, func(s string) bool {
		return s == ""
	})

	queryPlaceholders := []string{
		"db",
		collectionName,
		fmt.Sprintf("find(%s)", strings.Join(findPlaceholders, ", ")),
		parseCursorOperations(commandRaw),
	}

	// Remove empty parts from queryPlaceholders
	queryPlaceholders = slices.DeleteFunc(queryPlaceholders, func(s string) bool {
		return s == ""
	})

	return strings.Join(queryPlaceholders, ".")
}
