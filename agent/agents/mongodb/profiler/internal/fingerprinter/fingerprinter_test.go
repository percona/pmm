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

package fingerprinter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/fingerprinter"
	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/percona/pmm/agent/utils/mongo_fix"
)

const (
	MgoTimeoutDialInfo      = 5 * time.Second
	MgoTimeoutSessionSync   = 5 * time.Second
	MgoTimeoutSessionSocket = 5 * time.Second
)

func createQuery(dbName string, startTime time.Time) bson.M {
	return bson.M{
		"ns": bson.M{"$ne": dbName + ".system.profile"},
		"ts": bson.M{"$gt": startTime},
	}
}

func createIterator(ctx context.Context, collection *mongo.Collection, query bson.M) (*mongo.Cursor, error) {
	opts := options.Find().SetSort(bson.M{"$natural": 1}).SetCursorType(options.TailableAwait)
	return collection.Find(ctx, query, opts)
}

type ProfilerStatus struct {
	Was      int64 `bson:"was"`
	SlowMs   int64 `bson:"slowms"`
	GleStats struct {
		ElectionID string `bson:"electionId"`
		LastOpTime int64  `bson:"lastOpTime"`
	} `bson:"$gleStats"`
}

func createSession(dsn string, agentID string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), MgoTimeoutDialInfo)
	defer cancel()

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		return nil, err
	}

	opts = opts.
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetSocketTimeout(MgoTimeoutSessionSocket).
		SetAppName(fmt.Sprintf("QAN-mongodb-profiler-%s", agentID))

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func TestProfilerFingerprinter(t *testing.T) {
	t.Run("CheckWithRealDB", func(t *testing.T) {
		url := "mongodb://root:root-password@127.0.0.1:27017"
		dbName := "test_fingerprint"

		client, err := createSession(url, "pmm-agent")
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), MgoTimeoutSessionSync)
		defer cancel()
		_ = client.Database(dbName).Drop(ctx)
		defer client.Database(dbName).Drop(context.TODO()) //nolint:errcheck

		ps := ProfilerStatus{}
		err = client.Database("admin").RunCommand(ctx, primitive.M{"profile": -1}).Decode(&ps)
		defer func() { // restore profiler status
			client.Database("admin").RunCommand(ctx, primitive.D{{"profile", ps.Was}, {"slowms", ps.SlowMs}})
		}()

		// Enable profilling all queries (2, slowms = 0)
		res := client.Database("admin").RunCommand(ctx, primitive.D{{"profile", 2}, {"slowms", 0}})
		if res.Err() != nil {
			return
		}

		database := client.Database(dbName)
		_, err = database.Collection("test").InsertOne(ctx, bson.M{"id": 0, "name": "test", "value": 1, "time": time.Now()})
		assert.NoError(t, err)
		_, err = database.Collection("secondcollection").InsertOne(ctx, bson.M{"id": 0, "name": "sec", "value": 2})
		assert.NoError(t, err)
		database.Collection("test").FindOne(ctx, bson.M{"id": 0})
		database.Collection("test").FindOne(ctx, bson.M{"id": 1, "name": "test", "time": time.Now()})
		database.Collection("test").FindOneAndUpdate(ctx, bson.M{"id": 0}, bson.M{"$set": bson.M{"name": "new"}})
		database.Collection("test").FindOneAndDelete(ctx, bson.M{"id": 1})
		database.Collection("secondcollection").Find(ctx, bson.M{"name": "sec"}, options.Find().SetLimit(1).SetSort(bson.M{"id": -1})) //nolint:errcheck
		database.Collection("test").Aggregate(ctx,                                                                                     //nolint:errcheck
			[]bson.M{
				{
					"$match": bson.M{"id": 0, "time": bson.M{"$gt": time.Now().Add(-time.Hour)}},
				},
				{
					"$group": bson.M{"_id": "$id", "count": bson.M{"$sum": 1}},
				},
				{
					"$sort": bson.M{"_id": 1},
				},
			},
		)
		database.Collection("secondcollection").Aggregate(ctx, mongo.Pipeline{ //nolint:errcheck
			bson.D{
				{
					Key: "$collStats",
					Value: bson.M{
						// TODO: PMM-9568 : Add support to handle histogram metrics
						"latencyStats": bson.M{"histograms": false},
						"storageStats": bson.M{"scale": 1},
					},
				},
			}, bson.D{
				{
					Key: "$project",
					Value: bson.M{
						"storageStats.wiredTiger":   0,
						"storageStats.indexDetails": 0,
					},
				},
			},
		})
		database.Collection("secondcollection").DeleteOne(ctx, bson.M{"id": 0}) //nolint:errcheck
		database.Collection("test").DeleteMany(ctx, bson.M{"name": "test"})     //nolint:errcheck
		profilerCollection := database.Collection("system.profile")
		query := createQuery(dbName, time.Now().Add(-10*time.Minute))

		cursor, err := createIterator(ctx, profilerCollection, query)
		require.NoError(t, err)
		// do not cancel cursor closing when ctx is canceled
		defer cursor.Close(context.Background()) //nolint:errcheck

		pf := &ProfilerFingerprinter{}

		var fingerprints []string
		for cursor.TryNext(ctx) {
			doc := proto.SystemProfile{}
			e := cursor.Decode(&doc)
			require.NoError(t, e)

			b := bson.M{}
			e = cursor.Decode(&b)
			require.NoError(t, e)

			marshal, e := json.Marshal(b)
			require.NoError(t, e)
			log.Println(string(marshal))

			fingerprint, err := pf.Fingerprint(doc)
			require.NoError(t, err)
			require.NotNil(t, fingerprint)
			fingerprints = append(fingerprints, fingerprint.Fingerprint)
		}
		assert.NotEmpty(t, fingerprints)
		expectedFingerprints := []string{
			`db.test.insert(?)`,
			`db.secondcollection.insert(?)`,
			`db.test.find({"id":"?"}).limit(?)`,
			`db.test.find({"id":"?","name":"?","time":"?"}).limit(?)`,
			`db.runCommand({"findAndModify":"test","query":{"id":"?"},"update":{"$set":{"name":"?"}}})`,
			`db.runCommand({"findAndModify":"test","query":{"id":"?"},"remove":true})`,
			`db.secondcollection.find({"name":"?"}).sort({"id":-1}).limit(?)`,
			`db.test.aggregate([{"$match":{"id":"?","time":{"$gt":"?"}}}, {"$group":{"_id":"$id","count":{"$sum":1}}}, {"$sort":{"_id":1}}])`,
			`db.test.aggregate([{"$match":{"id":"?","time":{"$gt":"?"}}}, {"$group":{"count":{"$sum":1},"_id":"$id"}}, {"$sort":{"_id":1}}])`,
			`db.secondcollection.aggregate([{"$collStats":{"latencyStats":{"histograms":false},"storageStats":{"scale":1}}}, {"$project":{"storageStats.wiredTiger":0,"storageStats.indexDetails":0}}])`,
			`db.secondcollection.aggregate([{"$collStats":{"latencyStats":{"histograms":false},"storageStats":{"scale":1}}}, {"$project":{"storageStats.indexDetails":0,"storageStats.wiredTiger":0}}])`,
			`db.secondcollection.aggregate([{"$collStats":{"storageStats":{"scale":1},"latencyStats":{"histograms":false}}}, {"$project":{"storageStats.wiredTiger":0,"storageStats.indexDetails":0}}])`,
			`db.secondcollection.aggregate([{"$collStats":{"storageStats":{"scale":1},"latencyStats":{"histograms":false}}}, {"$project":{"storageStats.indexDetails":0,"storageStats.wiredTiger":0}}])`,
			`db.secondcollection.deleteOne({"id":"?"})`,
			`db.test.deleteMany({"name":"?"})`,
		}
		for i, fingerprint := range fingerprints {
			assert.Contains(t, expectedFingerprints, fingerprint, "fingerprint %d: %s", i, fingerprint)
		}
	})

	type testCase struct {
		name string
		doc  proto.SystemProfile
		want fingerprinter.Fingerprint
	}
	tests := []testCase{
		{
			name: "find",
			doc: proto.SystemProfile{
				Ns:      "test.collection",
				Op:      "query",
				Command: bson.D{{Key: "filter", Value: bson.D{{Key: "name", Value: "test"}}}, {Key: "sort", Value: bson.D{{Key: "_id", Value: 1}}}, {Key: "limit", Value: 4}, {Key: "skip", Value: 5}},
			},
			want: fingerprinter.Fingerprint{
				Fingerprint: `db.collection.find({"name":"?"}).sort({"_id":1}).limit(?).skip(?)`,
				Namespace:   "test.collection",
				Database:    "test",
				Collection:  "collection",
				Operation:   "query",
			},
		},
		{
			name: "insert",
			doc: proto.SystemProfile{
				Ns:      "test.insert_collection",
				Op:      "insert",
				Command: bson.D{},
			},
			want: fingerprinter.Fingerprint{
				Fingerprint: `db.insert_collection.insert(?)`,
				Namespace:   "test.insert_collection",
				Database:    "test",
				Collection:  "insert_collection",
				Operation:   "insert",
			},
		},
		{
			name: "update",
			doc: proto.SystemProfile{
				Ns:      "test.update_collection",
				Op:      "update",
				Command: bson.D{{Key: "q", Value: bson.D{{Key: "name", Value: "test"}}}, {Key: "u", Value: bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "new"}}}}}},
			},
			want: fingerprinter.Fingerprint{
				Fingerprint: `db.update_collection.update({"name":"?"}, {"$set":{"name":"?"}})`,
				Namespace:   "test.update_collection",
				Database:    "test",
				Collection:  "update_collection",
				Operation:   "update",
			},
		},
		{
			name: "update 8.0",
			doc: proto.SystemProfile{
				Op: "update",
				Ns: "config.system.sessions",
				Command: bson.D{
					{Key: "q", Value: bson.D{
						{Key: "_id", Value: bson.D{
							{Key: "id", Value: primitive.NewObjectID()},
							{Key: "uid", Value: primitive.Binary{Subtype: 0, Data: []byte{0x47, 0xDE, 0x50, 0x98, 0xFC, 0x1, 0x14, 0x9A, 0xFB, 0xF4, 0xC8, 0x99, 0x6F, 0xB9, 0x24, 0x27, 0xAE, 0x41, 0xE4, 0x64, 0x9B, 0x93, 0x4C, 0xA4, 0x95, 0x99, 0x1B, 0x78, 0x52, 0xB8, 0x55}}},
						}},
					}},
					{Key: "u", Value: bson.A{
						bson.D{{Key: "$set", Value: bson.D{{Key: "lastUse", Value: "$$NOW"}}}},
					}},
					{Key: "multi", Value: false},
					{Key: "upsert", Value: true},
				},
			},
			want: fingerprinter.Fingerprint{
				Fingerprint: `db.system.sessions.update({"_id":{"id":"?","uid":"?"}}, [{"$set":{"lastUse":"?"}}], {"upsert":true})`,
				Namespace:   "config.system.sessions",
				Database:    "config",
				Collection:  "system.sessions",
				Operation:   "update",
			},
		},
		{
			name: "delete",
			doc: proto.SystemProfile{
				Ns:      "test.delete_collection",
				Op:      "remove",
				Command: bson.D{{Key: "q", Value: bson.D{{Key: "name", Value: "test"}}}},
			},
			want: fingerprinter.Fingerprint{
				Fingerprint: `db.delete_collection.deleteMany({"name":"?"})`,
				Namespace:   "test.delete_collection",
				Database:    "test",
				Collection:  "delete_collection",
				Operation:   "remove",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf := &ProfilerFingerprinter{}
			fingerprint, err := pf.Fingerprint(tt.doc)
			require.NoError(t, err)
			require.NotNil(t, fingerprint)
			assert.Equal(t, tt.want.Fingerprint, fingerprint.Fingerprint)
			assert.Equal(t, tt.want.Namespace, fingerprint.Namespace)
			assert.Equal(t, tt.want.Database, fingerprint.Database)
			assert.Equal(t, tt.want.Collection, fingerprint.Collection)
			assert.Equal(t, tt.want.Operation, fingerprint.Operation)
		})
	}
}
