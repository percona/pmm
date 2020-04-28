// pmm-agent
// Copyright 2019 Percona LLC
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

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongodbExplainAction struct {
	id     string
	params *agentpb.StartActionRequest_MongoDBExplainParams
}

// NewMongoDBExplainAction creates a MongoDB EXPLAIN query Action.
func NewMongoDBExplainAction(id string, params *agentpb.StartActionRequest_MongoDBExplainParams) Action {
	return &mongodbExplainAction{
		id:     id,
		params: params,
	}
}

// ID returns an Action ID.
func (a *mongodbExplainAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *mongodbExplainAction) Type() string {
	return "mongodb-explain"
}

// Run runs an Action and returns output and error.
func (a *mongodbExplainAction) Run(ctx context.Context) ([]byte, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(a.params.Dsn))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	var eq proto.ExampleQuery

	err = bson.UnmarshalExtJSON([]byte(a.params.Query), true, &eq)
	if err != nil {
		return nil, fmt.Errorf("explain: unable to decode query %s: %s", a.params.Query, err)
	}

	res := client.Database("admin").RunCommand(ctx, eq.ExplainCmd())
	if res.Err() != nil {
		return nil, res.Err()
	}

	result, err := res.DecodeBytes()
	if err != nil {
		return nil, err
	}
	// We need it because result
	return []byte(result.String()), nil
}

func (a *mongodbExplainAction) sealed() {}
