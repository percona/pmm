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
	"path/filepath"
	"strings"

	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/percona/pmm-agent/utils/templates"
)

// MongoDBQueryAdmincommandActionParams represent Mongo DB Query Admin Command Action params.
type MongoDBQueryAdmincommandActionParams struct {
	ID      string
	DSN     string
	Files   *agentpb.TextFiles
	Command string
	Arg     interface{}
	TempDir string
}

type mongodbQueryAdmincommandAction struct {
	id      string
	dsn     string
	files   *agentpb.TextFiles
	command string
	arg     interface{}
	tempDir string
}

// NewMongoDBQueryAdmincommandAction creates a MongoDB adminCommand query Action.
func NewMongoDBQueryAdmincommandAction(params MongoDBQueryAdmincommandActionParams) Action {
	return &mongodbQueryAdmincommandAction{
		id:      params.ID,
		dsn:     params.DSN,
		files:   params.Files,
		command: params.Command,
		arg:     params.Arg,
		tempDir: params.TempDir,
	}
}

// ID returns an Action ID.
func (a *mongodbQueryAdmincommandAction) ID() string {
	return a.id
}

// Type returns an Action type.
func (a *mongodbQueryAdmincommandAction) Type() string {
	return "mongodb-query-admincommand"
}

// Run runs an Action and returns output and error.
func (a *mongodbQueryAdmincommandAction) Run(ctx context.Context) ([]byte, error) {
	dsn, err := templates.RenderDSN(a.dsn, a.files, filepath.Join(a.tempDir, strings.ToLower(a.Type()), a.id))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer client.Disconnect(ctx) //nolint:errcheck

	runCommand := bson.D{{a.command, a.arg}} //nolint:govet
	res := client.Database("admin").RunCommand(ctx, runCommand)

	var doc map[string]interface{}
	if err = res.Decode(&doc); err != nil {
		return nil, errors.WithStack(err)
	}

	data := []map[string]interface{}{doc}
	return agentpb.MarshalActionQueryDocsResult(data)
}

func (a *mongodbQueryAdmincommandAction) sealed() {}
