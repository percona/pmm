// Copyright 2023 Percona LLC
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

package version

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/version"
)

// GetMongoDBVersion returns the parsed version of the connected MongoDB server.
func GetMongoDBVersion(ctx context.Context, client *mongo.Client) (*version.Parsed, error) {
	resp := client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}})
	if err := resp.Err(); err != nil {
		return nil, err
	}

	buildInfo := struct {
		Version string `bson:"version"`
	}{}

	if err := resp.Decode(&buildInfo); err != nil {
		return nil, err
	}

	mongoVersion, err := version.Parse(buildInfo.Version)
	if err != nil {
		return nil, err
	}
	return mongoVersion, nil
}
