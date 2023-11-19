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

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/utils/mongo_fix"
	"github.com/percona/pmm/agent/utils/templates"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

type mongodbExplainAction struct {
	id      string
	timeout time.Duration
	params  *agentv1.StartActionRequest_MongoDBExplainParams
	tempDir string
}

var errCannotExplain = fmt.Errorf("cannot explain this type of query")

// NewMongoDBExplainAction creates a MongoDB EXPLAIN query Action.
func NewMongoDBExplainAction(id string, timeout time.Duration, params *agentv1.StartActionRequest_MongoDBExplainParams, tempDir string) Action {
	return &mongodbExplainAction{
		id:      id,
		timeout: timeout,
		params:  params,
		tempDir: tempDir,
	}
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
	return "mongodb-explain"
}

// Run runs an action and returns output and error.
func (a *mongodbExplainAction) Run(ctx context.Context) ([]byte, error) {
	dsn, err := templates.RenderDSN(a.params.Dsn, a.params.TextFiles, filepath.Join(a.tempDir, strings.ToLower(a.Type()), a.id))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

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

	database := "admin"
	if eq.Db() != "" {
		database = eq.Db()
	}
	res := client.Database(database).RunCommand(ctx, eq.ExplainCmd())
	if res.Err() != nil {
		return nil, errors.Wrap(errCannotExplain, res.Err().Error())
	}

	result, err := res.DecodeBytes()
	if err != nil {
		return nil, err
	}
	// We need it because result
	return []byte(result.String()), nil
}

func (a *mongodbExplainAction) sealed() {}
