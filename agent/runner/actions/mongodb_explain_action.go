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
	"go.mongodb.org/mongo-driver/mongo"

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

// Run runs an action and returns output and error.
func (a *mongodbExplainAction) Run(ctx context.Context) ([]byte, error) {
	opts, err := mongo_fix.ClientOptionsForDSN(a.dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	if notMinMongoDBVersion(ctx, client) {
		version := strings.TrimSuffix(minMongoDBVersion.String(), "-0")
		err := fmt.Sprintf("minimum supported version is %s, please update your MongoDB", version)
		return nil, errors.New(err)
	}

	return explainForQuery(ctx, client, a.params.Query)
}

func (a *mongodbExplainAction) sealed() {}
