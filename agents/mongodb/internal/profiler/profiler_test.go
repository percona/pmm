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

package profiler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"

	"github.com/percona/pmm-agent/agents/mongodb/internal/profiler/aggregator"
	"github.com/percona/pmm-agent/agents/mongodb/internal/report"
)

type MongoVersion struct {
	VersionString string `bson:"version"`
	PSMDBVersion  string `bson:"psmdbVersion"`
	Version       []int  `bson:"versionArray"`
}

func GetMongoVersion(ctx context.Context, client *mongo.Client) (string, error) {
	ver := new(MongoVersion)
	err := client.Database("admin").RunCommand(ctx, bson.D{{"buildInfo", 1}}).Decode(ver)
	if err != nil {
		return "", nil
	}

	version := fmt.Sprintf("%d.%d", ver.Version[0], ver.Version[1])
	return version, err
}

func TestProfiler(t *testing.T) {
	defaultInterval := aggregator.DefaultInterval
	aggregator.DefaultInterval = time.Second
	defer func() { aggregator.DefaultInterval = defaultInterval }()

	url := "mongodb://root:root-password@127.0.0.1:27017"

	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(logrus.InfoLevel)

	sess, err := createSession(url, "pmm-agent")
	require.NoError(t, err)

	cleanUpDBs(t, sess) // Just in case there are old dbs with matching names

	dbsCount := 10
	docsCount := float32(10)

	ticker := time.NewTicker(time.Millisecond)
	i := 0
	// It's done to create Databases.
	for i < dbsCount {
		<-ticker.C
		doc := bson.M{"id": i}
		dbName := fmt.Sprintf("test_%02d", i)
		logrus.Traceln("create db", dbName)
		_, err := sess.Database(dbName).Collection("test").InsertOne(context.TODO(), doc)
		assert.NoError(t, err)
		i++
	}
	<-time.After(aggregator.DefaultInterval) // give it some time before starting profiler

	ms := &testWriter{
		t:       t,
		reports: make([]*report.Report, 0),
	}
	prof := New(url, logrus.WithField("component", "profiler-test"), ms, "test-id")
	err = prof.Start()
	defer prof.Stop()
	require.NoError(t, err)
	<-time.After(aggregator.DefaultInterval * 2) // give it some time to start profiler

	i = 0
	for i < dbsCount*int(docsCount) {
		<-ticker.C
		dbNumber := i / int(docsCount)
		fieldsCount := dbNumber + 1
		doc := bson.M{}
		for j := 0; j < fieldsCount; j++ {
			doc[fmt.Sprintf("name_%05d", j)] = fmt.Sprintf("value_%05d", j) // to generate different fingerprints
		}
		dbName := fmt.Sprintf("test_%02d", dbNumber)
		logrus.Tracef("inserting value %d to %s", i, dbName)
		_, err = sess.Database(dbName).Collection("people").InsertOne(context.TODO(), doc)
		assert.NoError(t, err)
		i++
	}
	<-time.After(aggregator.DefaultInterval * 6) // give it some time to catch all metrics

	err = prof.Stop()
	require.NoError(t, err)

	defer cleanUpDBs(t, sess)

	require.GreaterOrEqual(t, len(ms.reports), 1)

	buckets := make(map[string]*agentpb.MetricsBucket)
	for _, r := range ms.reports {
		for _, bucket := range r.Buckets {
			if bucket.Common.Fingerprint != "INSERT people" {
				continue
			}
			key := fmt.Sprintf("%s:%s", bucket.Common.Database, bucket.Common.Fingerprint)
			if b, ok := buckets[key]; ok {
				b.Mongodb.MDocsReturnedCnt += bucket.Mongodb.MDocsReturnedCnt
				b.Mongodb.MResponseLengthCnt += bucket.Mongodb.MResponseLengthCnt
				b.Mongodb.MResponseLengthSum += bucket.Mongodb.MResponseLengthSum
				b.Mongodb.MDocsScannedCnt += bucket.Mongodb.MDocsScannedCnt
			} else {
				buckets[key] = bucket
			}
		}
	}

	version, err := GetMongoVersion(context.TODO(), sess)
	require.NoError(t, err)

	var responseLength float32
	switch version {
	case "3.4":
		responseLength = 44
	case "3.6":
		responseLength = 29
	default:
		responseLength = 45
	}

	assert.Equal(t, dbsCount, len(buckets)) // 300 sample docs / 10 = different database names
	for _, bucket := range buckets {
		assert.True(t, strings.HasPrefix(bucket.Common.Database, "test_"), fmt.Sprintf("database name %s should have prefix test_", bucket.Common.Database))
		assert.Equal(t, "INSERT people", bucket.Common.Fingerprint)
		assert.Equal(t, []string{"people"}, bucket.Common.Tables)
		assert.Equal(t, "test-id", bucket.Common.AgentId)
		assert.Equal(t, inventorypb.AgentType(9), bucket.Common.AgentType)
		wantMongoDB := &agentpb.MetricsBucket_MongoDB{
			MDocsReturnedCnt:   docsCount,
			MResponseLengthCnt: docsCount,
			MResponseLengthSum: responseLength * docsCount,
			MResponseLengthMin: responseLength,
			MResponseLengthMax: responseLength,
			MResponseLengthP99: responseLength,
			MDocsScannedCnt:    docsCount,
		}
		assert.Equalf(t, wantMongoDB, bucket.Mongodb, "wrong metrics for db %s", bucket.Common.Database)
	}
}

func cleanUpDBs(t *testing.T, sess *mongo.Client) {
	dbs, err := sess.ListDatabaseNames(context.TODO(), bson.M{})
	for _, dbname := range dbs {
		if strings.HasPrefix("test_", dbname) {
			err = sess.Database(dbname).Drop(context.TODO())
			require.NoError(t, err)
		}
	}
}

type testWriter struct {
	t       *testing.T
	reports []*report.Report
}

func (tw *testWriter) Write(actual *report.Report) error {
	require.NotNil(tw.t, actual)
	tw.reports = append(tw.reports, actual)
	return nil
}
