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

package realtimeanalytics

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/percona/pmm/agent/utils/mongofix"
)

const (
	// Timeout for establishing connection to MongoDB.
	mgoConnectTimeout = 5 * time.Second
	// Timeout for MongoDB queries.
	mgoQueryTimeout = 5 * time.Second
	// Timeout for MongoDB session socket.
	mgoTimeoutSessionSocket = 5 * time.Second
)

// createSession creates new MongoDB client and checks connection to MongoDB by pinging it.
func createSession(ctx context.Context, dsn string, agentID string) (*mongo.Client, error) {
	// if dsn is incorrect we should exit immediately as this is not gonna correct itself
	opts, err := mongofix.ClientOptionsForDSN(dsn)
	if err != nil {
		return nil, err
	}

	opts = opts.
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetTimeout(mgoQueryTimeout).
		SetConnectTimeout(mgoConnectTimeout).
		SetSocketTimeout(mgoTimeoutSessionSocket).
		SetCompressors([]string{"snappy", "zlib", "zstd"}).
		SetAppName("rta-mongodb-" + agentID)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Nearest())
	if err != nil {
		return nil, err
	}

	return client, nil
}
