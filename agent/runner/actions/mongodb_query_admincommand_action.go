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
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/utils/mongo_fix"
	"github.com/percona/pmm/agent/utils/templates"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const mongoDBQueryAdminCommandActionType = "mongodb-query-admincommand"

type mongodbQueryAdmincommandAction struct {
	id      string
	timeout time.Duration
	dsn     string
	files   *agentv1.TextFiles //nolint:unused
	command string
	arg     interface{}
	tmpDir  string
}

// NewMongoDBQueryAdmincommandAction creates a MongoDB adminCommand query action.
func NewMongoDBQueryAdmincommandAction(
	id string,
	timeout time.Duration,
	dsn string,
	files *agentv1.TextFiles,
	command string,
	arg interface{},
	tempDir string,
) (Action, error) {
	tmpDir := filepath.Join(tempDir, mongoDBQueryAdminCommandActionType, id)
	dsn, err := templates.RenderDSN(dsn, files, tmpDir)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &mongodbQueryAdmincommandAction{
		id:      id,
		timeout: timeout,
		dsn:     dsn,
		command: command,
		arg:     arg,
		tmpDir:  tmpDir,
	}, nil
}

// ID returns an action ID.
func (a *mongodbQueryAdmincommandAction) ID() string {
	return a.id
}

// Timeout returns Action timeout.
func (a *mongodbQueryAdmincommandAction) Timeout() time.Duration {
	return a.timeout
}

// Type returns an action type.
func (a *mongodbQueryAdmincommandAction) Type() string {
	return mongoDBQueryAdminCommandActionType
}

// DSN returns a DSN for the Action.
func (a *mongodbQueryAdmincommandAction) DSN() string {
	return a.dsn
}

// Run runs an action and returns output and error.
func (a *mongodbQueryAdmincommandAction) Run(ctx context.Context) ([]byte, error) {
	defer templates.CleanupTempDir(a.tmpDir, logrus.WithField("component", mongoDBQueryAdminCommandActionType))
	opts, err := mongo_fix.ClientOptionsForDSN(a.dsn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := mongo.Connect(ctx, opts)
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
	return agentv1.MarshalActionQueryDocsResult(data)
}

func (a *mongodbQueryAdmincommandAction) sealed() {}
