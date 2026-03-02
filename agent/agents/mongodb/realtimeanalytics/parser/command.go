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

func parseCommand(raw bson.Raw) string {
	commandRaw, _ := raw.Lookup("command").DocumentOK()

	if aggregate, ok := commandRaw.Lookup("aggregate").Int32OK(); ok && aggregate == 1 {
		ns, _ := raw.Lookup("ns").StringValueOK()
		return parseCommandAggregate(commandRaw, ns)
	}

	commandPlaceholders := []string{
		fmt.Sprintf(`db.runCommand(%s)`, parseDocument(raw, "command")),
	}
	if cursorOperations := parseCursorOperations(commandRaw); cursorOperations != "" {
		commandPlaceholders = append(commandPlaceholders, cursorOperations)
	}

	return strings.Join(commandPlaceholders, ".")
}
