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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/percona/pmm/agent/utils/mongo_fix"
	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/api/agentpb"
)

const mongoDBExplainActionType = "mongodb-explain"

type mongodbExplainAction struct {
	id      string
	timeout time.Duration
	params  *agentpb.StartActionRequest_MongoDBExplainParams
	dsn     string
}

var errCannotExplain = fmt.Errorf("cannot explain this type of query")

// NewMongoDBExplainAction creates a MongoDB EXPLAIN query Action.
func NewMongoDBExplainAction(id string, timeout time.Duration, params *agentpb.StartActionRequest_MongoDBExplainParams, tempDir string) (Action, error) {
	dsn, err := templates.RenderDSN(params.Dsn, params.TextFiles, filepath.Join(tempDir, mongoDBExplainActionType, id))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &mongodbExplainAction{
		id:      id,
		timeout: timeout,
		params:  params,
		dsn:     dsn,
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

type ExampleQuery2 struct {
	Ns                 string `bson:"ns" json:"ns"`
	Op                 string `bson:"op" json:"op"`
	Query              bson.D `bson:"query,omitempty" json:"query,omitempty"`
	Command            bson.D `bson:"command,omitempty" json:"command,omitempty"`
	OriginatingCommand bson.D `bson:"originatingCommand,omitempty" json:"originatingCommand,omitempty"`
	UpdateObj          bson.D `bson:"updateobj,omitempty" json:"updateobj,omitempty"`
}

func (self ExampleQuery2) Db() string {
	ns := strings.SplitN(self.Ns, ".", 2)
	if len(ns) > 0 {
		return ns[0]
	}
	return ""
}

// Run runs an action and returns output and error.
func (a *mongodbExplainAction) Run(ctx context.Context) ([]byte, error) {
	opts, err := mongo_fix.ClientOptionsForDSN(a.dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// opts.BSONOptions = &options.BSONOptions{
	// 	UseJSONStructTags:   true,
	// 	NilSliceAsEmpty:     true,
	// 	NilMapAsEmpty:       true,
	// 	NilByteSliceAsEmpty: true,
	// }

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	var eq proto.ExampleQuery
	err = bson.UnmarshalExtJSON([]byte(a.params.Query), true, &eq)
	if err != nil {
		return nil, errors.Wrapf(err, "Query: %s", a.params.Query)
	}

	fmt.Println(eq)

	database := "admin"
	if eq.Db() != "" {
		database = eq.Db()
	}
	command := bson.D{{"explain", eq.Command}}
	res := client.Database(database).RunCommand(ctx, command)
	if res.Err() != nil {
		return nil, errors.Wrap(errCannotExplain, res.Err().Error())
	}

	result, err := res.Raw()
	if err != nil {
		return nil, err
	}

	return []byte(result.String()), nil
}

func (a *mongodbExplainAction) sealed() {}
