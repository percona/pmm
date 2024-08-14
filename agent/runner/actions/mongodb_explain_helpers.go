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

package actions

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/version"
)

var minMongoDBVersion = version.MustParse("4.0.0-0")

type explain struct {
	Ns                 string `json:"ns"`
	Op                 string `json:"op"`
	Query              bson.D `json:"query,omitempty"`
	Command            bson.D `json:"command,omitempty"`
	OriginatingCommand bson.D `json:"originatingCommand,omitempty"`
	UpdateObj          bson.D `json:"updateobj,omitempty"`
}

func (e explain) prepareCommand() bson.D {
	command := e.Command
	switch e.Op {
	case "query":
		if len(command) == 0 {
			command = e.Query
		}

		if len(command) == 0 || command[0].Key != "find" {
			var filter any
			if len(command) != 0 && command[0].Key == "query" {
				filter = command[0].Value
			} else {
				filter = command
			}

			command = bson.D{
				{Key: "find", Value: e.getCollection()},
				{Key: "filter", Value: filter},
			}
		}
	case "update":
		if len(command) == 0 {
			command = bson.D{
				{Key: "q", Value: e.Query},
				{Key: "u", Value: e.UpdateObj},
			}
		}

		return bson.D{
			{Key: "update", Value: e.getCollection()},
			{Key: "updates", Value: []any{command}},
		}
	case "remove":
		if len(command) == 0 {
			command = bson.D{{Key: "q", Value: e.Query}}
		}

		return bson.D{
			{Key: "delete", Value: e.getCollection()},
			{Key: "deletes", Value: []any{command}},
		}
	case "insert":
		if len(command) == 0 {
			command = e.Query
		}
		if len(command) == 0 || command[0].Key != "insert" {
			command = bson.D{{Key: "insert", Value: e.getCollection()}}
		}
	case "getmore":
		if len(e.OriginatingCommand) == 0 {
			command = bson.D{{Key: "getmore", Value: ""}}
		}
	case "command":
		if len(command) == 0 || command[0].Key != "group" {
			break
		}

		command = fixReduceField(command)
	}

	return command
}

func (e explain) getDB() string {
	s := strings.SplitN(e.Ns, ".", 2)
	if len(s) == 2 {
		return s[0]
	}

	return ""
}

func (e explain) getCollection() string {
	s := strings.SplitN(e.Ns, ".", 2)
	if len(s) == 2 {
		return s[1]
	}

	return ""
}

func fixReduceField(command bson.D) bson.D {
	var group bson.D
	var ok bool
	if group, ok = command[0].Value.(bson.D); !ok {
		return command
	}

	for i := range group {
		if group[i].Key == "$reduce" {
			group[i].Value = "{}"
			command[0].Value = group
			break
		}
	}

	return command
}

func explainForQuery(ctx context.Context, client *mongo.Client, query string) ([]byte, error) {
	var e explain
	err := bson.UnmarshalExtJSON([]byte(query), true, &e)
	if err != nil {
		return nil, errors.Wrapf(err, "Query: %s", query)
	}

	command := bson.D{{Key: "explain", Value: e.prepareCommand()}}
	res := client.Database(e.getDB()).RunCommand(ctx, command)
	if res.Err() != nil {
		return nil, errors.Wrap(errCannotExplain, res.Err().Error())
	}

	result, err := res.Raw()
	if err != nil {
		return nil, err
	}

	return []byte(result.String()), nil
}

func notMinMongoDBVersion(ctx context.Context, client *mongo.Client) bool {
	var result bson.M
	command := bson.D{{Key: "serverStatus", Value: 1}}
	err := client.Database("check").RunCommand(ctx, command).Decode(&result)
	if err != nil || result["version"] == nil {
		return false
	}

	var dbVersion string
	var ok bool
	if dbVersion, ok = result["version"].(string); !ok {
		return false
	}
	currentVersion, err := version.Parse(dbVersion)
	if err != nil {
		return false
	}

	return currentVersion.Less(minMongoDBVersion)
}
