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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/utils/mongo_fix"
	"github.com/percona/pmm/agent/utils/templates"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const mongoDBExplainActionType = "mongodb-explain"

type mongodbExplainAction struct {
	id      string
	timeout time.Duration
	params  *agentv1.StartActionRequest_MongoDBExplainParams
	dsn     string
	tmpDir  string
}

type explain struct {
	Ns                 string `json:"ns"`
	Op                 string `json:"op"`
	Query              bson.D `json:"query,omitempty"`
	Command            bson.D `json:"command,omitempty"`
	OriginatingCommand bson.D `json:"originatingCommand,omitempty"`
	UpdateObj          bson.D `json:"updateobj,omitempty"`
}

var errCannotExplain = fmt.Errorf("cannot explain this type of query")

// NewMongoDBExplainAction creates a MongoDB EXPLAIN query Action.
func NewMongoDBExplainAction(id string, timeout time.Duration, params *agentv1.StartActionRequest_MongoDBExplainParams, tempDir string) (Action, error) {
	tmpDir := filepath.Join(tempDir, mongoDBExplainActionType, id)
	dsn, err := templates.RenderDSN(params.Dsn, params.TextFiles, tmpDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &mongodbExplainAction{
		id:      id,
		timeout: timeout,
		params:  params,
		dsn:     dsn,
		tmpDir:  tmpDir,
	}, nil
}

// ID returns an Action ID.
func (a *mongodbExplainAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *mongodbExplainAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an Action type.
func (a *mongodbExplainAction) Type() string {
	return mongoDBExplainActionType
}

// DSN returns the DSN for the Action.
func (a *mongodbExplainAction) DSN() string {
	return a.dsn
}

// Run runs an action and returns output and error.
func (a *mongodbExplainAction) Run(ctx context.Context) ([]byte, error) {
	defer templates.CleanupTempDir(a.tmpDir, logrus.WithField("component", mongoDBExplainActionType))

	opts, err := mongo_fix.ClientOptionsForDSN(a.dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	return explainForQuery(ctx, client, a.params.Query)
}

func (a *mongodbExplainAction) sealed() {}

func (e explain) prepareCommand() (bson.D, error) {
	command := e.Command

	switch e.Op {
	case "query":
		if len(command) == 0 {
			command = e.Query
		}

		if len(command) == 0 || command[0].Key != "find" {
			return bson.D{
				{Key: "find", Value: e.getCollection()},
				{Key: "filter", Value: command},
			}, nil
		}

		if len(command) != 0 && command[0].Key == "query" {
			return bson.D{
				{Key: "find", Value: e.getCollection()},
				{Key: "filter", Value: command[0].Value},
			}, nil
		}

		return dropDBField(command), nil
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
		}, nil
	case "remove":
		if len(command) == 0 {
			command = bson.D{{Key: "q", Value: e.Query}}
		}

		return bson.D{
			{Key: "delete", Value: e.getCollection()},
			{Key: "deletes", Value: []any{command}},
		}, nil
	case "getmore":
		if len(e.OriginatingCommand) == 0 {
			return bson.D{{Key: "getmore", Value: ""}}, nil
		}

		command = e.OriginatingCommand

		return dropDBField(command), nil
	case "command":
		command = dropDBField(command)

		if len(command) == 0 {
			return command, nil
		}

		switch command[0].Key {
		// Not supported commands.
		case "dbStats":
			return nil, errors.Errorf("command %s is not supported for explain", command[0].Key)
		case "group":
		default:
			// https://www.mongodb.com/docs/manual/tutorial/use-database-commands/?utm_source=chatgpt.com#database-command-form
			return reorderToCommandFirst(command), nil
		}

		return fixReduceField(command), nil
	// Not supported operations.
	case "insert", "drop":
		return nil, errors.Errorf("operation %s is not supported for explain", e.Op)
	}

	return command, nil
}

func reorderToCommandFirst(doc bson.D) bson.D {
	recognized := map[string]struct{}{
		"find": {}, "findandmodify": {}, "insert": {}, "update": {}, "delete": {},
		"aggregate": {}, "count": {}, "distinct": {}, "mapReduce": {},
		"collStats": {}, "listIndexes": {}, "currentOp": {}, "explain": {},
		"getMore": {}, "killCursors": {}, "create": {}, "drop": {},
		"listCollections": {}, "listDatabases": {}, "validate": {},
	}

	var first bson.E
	rest := []bson.E{}
	for _, e := range doc {
		if _, ok := recognized[e.Key]; ok && first.Key == "" {
			first = e
			continue
		}

		rest = append(rest, e)
	}

	if first.Key != "" {
		return append(bson.D{first}, rest...)
	}

	return doc
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

// dropDBField remove DB field to be able run explain on all supported types.
// Otherwise it could end up with BSON field 'xxx.$db' is a duplicate field.
func dropDBField(command bson.D) bson.D {
	for i := range command {
		if command[i].Key != "$db" {
			continue
		}

		if len(command)-1 == i {
			return command[:i]
		}

		return append(command[:i], command[i+1:]...)
	}

	return command
}

// fixReduceField fixing nil/empty values after unmarshalling funcs.
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
	err := bson.UnmarshalExtJSON([]byte(query), false, &e)
	if err != nil {
		return nil, errors.Wrapf(err, "Query: %s", query)
	}

	preparedCommand, err := e.prepareCommand()
	if err != nil {
		return nil, errors.Wrap(errCannotExplain, err.Error())
	}
	command := bson.D{{Key: "explain", Value: preparedCommand}}
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
