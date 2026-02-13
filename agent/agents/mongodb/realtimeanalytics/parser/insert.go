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

	"go.mongodb.org/mongo-driver/v2/bson"
)

var insertOptions = []string{"ordered", "writeConcern"}

func parseCommandInsert(commandRaw bson.Raw) string {
	collectionName, _ := commandRaw.Lookup("insert").StringValueOK()

	// command has format:
	// https://www.mongodb.com/docs/manual/reference/method/db.collection.insert/#syntax
	// db.%s.insert(<document | [<document1>, <document2>, ...]>, <{ ordered: <boolean>, writeConcern: <document> }>)
	insertPlaceholders := []string{"?"}
	if params := parseOptions(commandRaw, insertOptions); params != "" {
		insertPlaceholders = append(insertPlaceholders, params)
	}

	queryPlaceholders := []string{
		"db",
		collectionName,
		fmt.Sprintf("insert(%s)", strings.Join(insertPlaceholders, ", ")),
	}

	return strings.Join(queryPlaceholders, ".")
}
