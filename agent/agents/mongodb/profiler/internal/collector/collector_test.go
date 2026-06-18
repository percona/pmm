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

package collector

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/percona/pmm/agent/utils/mongo_fix"
)

const (
	mgoTimeoutDialInfo      = 5 * time.Second
	mgoTimeoutSessionSync   = 5 * time.Second
	mgoTimeoutSessionSocket = 5 * time.Second
)

type ProfilerStatus struct {
	Was      int64 `bson:"was"`
	SlowMs   int64 `bson:"slowms"`
	GleStats struct {
		ElectionID string `bson:"electionId"`
		LastOpTime int64  `bson:"lastOpTime"`
	} `bson:"$gleStats"`
}

func BenchmarkCollector(b *testing.B) {
	maxLoops := 3
	maxDocs := 100

	timeout := time.Millisecond*time.Duration(maxDocs*maxLoops) + cursorTimeout*time.Duration(maxLoops*2) + time.Second

	ctx, cancel := context.WithTimeout(b.Context(), timeout)
	defer cancel()

	url := "mongodb://root:root-password@127.0.0.1:27017"
	// time.Millisecond*time.Duration(maxDocs*maxLoops): time it takes to write all docs for all iterations
	// cursorTimeout*time.Duration(maxLoops*2): Wait time between loops to produce iter.TryNext to return a false

	client, err := createSession(url, "pmm-agent")
	require.NoError(b, err)

	// Just in case there are old dbs with matching names
	require.NoError(b, cleanUpDBs(b.Context(), b, client))
	b.Cleanup(func() {
		cleanupCtx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCtx()
		assert.NoError(b, cleanUpDBs(cleanupCtx, b, client))
	})

	var ps ProfilerStatus
	err = client.Database("admin").RunCommand(ctx, primitive.M{"profile": -1}).Decode(&ps)
	require.NoError(b, err)
	b.Cleanup(func() { // restore profiler status
		cmdCtx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancelCtx()
		client.Database("admin").RunCommand(cmdCtx, primitive.D{{Key: "profile", Value: ps.Was}, {Key: "slowms", Value: ps.SlowMs}})
	})

	// Enable profilling all queries (2, slowms = 0)
	res := client.Database("admin").RunCommand(ctx, primitive.D{{Key: "profile", Value: 2}, {Key: "slowms", Value: 0}})
	require.NoError(b, res.Err())

	for b.Loop() {
		ctr := New(client, "test", logrus.WithField("component", "profiler-test"))
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go genData(ctx, client, maxLoops, maxDocs)

		var profiles []proto.SystemProfile
		docsChan, err := ctr.Start(ctx)
		if err != nil {
			return
		}

		go func() {
			i := 0
			for profile := range docsChan {
				profiles = append(profiles, profile)
				i++
				if i >= 300 {
					wg.Done()
				}
			}
		}()
		wg.Wait()
		ctr.Stop()
	}

	cancel()
}

func TestCollector(t *testing.T) {
	maxLoops := 3
	maxDocs := 100

	url := "mongodb://root:root-password@127.0.0.1:27017"
	// time.Millisecond*time.Duration(maxDocs*maxLoops): time it takes to write all docs for all iterations
	// cursorTimeout*time.Duration(maxLoops*2): Wait time between loops to produce iter.TryNext to return a false
	timeout := time.Millisecond*time.Duration(maxDocs*maxLoops) + cursorTimeout*time.Duration(maxLoops*2) + 5*time.Second

	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()

	client, err := createSession(url, "pmm-agent")
	require.NoError(t, err)

	require.NoError(t, cleanUpDBs(t.Context(), t, client)) // Just in case there are old dbs with matching names
	t.Cleanup(func() {
		cleanupCtx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCtx()
		assert.NoError(t, cleanUpDBs(cleanupCtx, t, client))
	})

	// It's done create DB before the test.
	doc := bson.M{}
	_, err = client.Database("test_collector").Collection("test").InsertOne(t.Context(), doc)
	require.NoError(t, err)
	<-time.After(time.Second)

	ctr := New(client, "test_collector", logrus.WithField("component", "collector-test"))

	// Start the collector
	var profiles []proto.SystemProfile
	docsChan, err := ctr.Start(ctx)
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	<-time.After(time.Second)

	go genData(ctx, client, maxLoops, maxDocs)

	go func() {
		defer wg.Done()
		i := 0
		for profile := range docsChan {
			select {
			case <-ctx.Done():
				return
			default:
			}
			profiles = append(profiles, profile)
			i++
			if i >= 300 {
				return
			}
		}
	}()

	wg.Wait()
	ctr.Stop()

	assert.Len(t, profiles, maxDocs*maxLoops)
}

func genData(ctx context.Context, client *mongo.Client, maxLoops, maxDocs int) {
	interval := time.Millisecond

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range maxLoops {
		select {
		case <-ctx.Done():
			return
		default:
		}

		for range maxDocs {
			select {
			case <-ticker.C:
				doc := bson.M{"first_name": "zapp", "last_name": "brannigan"}
				client.Database("test_collector").Collection("people").InsertOne(context.TODO(), doc)
			case <-ctx.Done():
				return
			}
		}

		<-time.After(cursorTimeout)
	}
}

func createSession(dsn string, agentID string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mgoTimeoutDialInfo)
	defer cancel()

	opts, err := mongo_fix.ClientOptionsForDSN(dsn)
	if err != nil {
		return nil, err
	}

	opts = opts.
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetSocketTimeout(mgoTimeoutSessionSocket).
		SetAppName("QAN-mongodb-profiler-" + agentID)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func cleanUpDBs[T testing.TB](ctx context.Context, t T, sess *mongo.Client) error {
	t.Helper()
	dbs, err := sess.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return err
	}
	for _, dbname := range dbs {
		if strings.HasPrefix(dbname, "test_") {
			err = sess.Database(dbname).Drop(ctx)
			if err != nil {
				t.Logf("failed to drop database %q: %v", dbname, err)
				continue
			}
		}
	}
	return nil
}
