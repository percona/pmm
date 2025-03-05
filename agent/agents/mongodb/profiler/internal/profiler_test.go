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

package profiler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/agents/mongodb/profiler/internal/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/profiler/internal/report"
	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

type MongoVersion struct {
	VersionString string `bson:"version"`
	PSMDBVersion  string `bson:"psmdbVersion"`
	Version       []int  `bson:"versionArray"`
}

func GetMongoVersion(ctx context.Context, client *mongo.Client) (string, error) {
	var ver MongoVersion
	err := client.Database("admin").RunCommand(ctx, bson.D{{"buildInfo", 1}}).Decode(&ver)
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

	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(logrus.InfoLevel)

	sslDSNTemplate, files := tests.GetTestMongoDBWithSSLDSN(t, "../../../../")
	tempDir := t.TempDir()
	sslDSN, err := templates.RenderDSN(sslDSNTemplate, files, tempDir)
	require.NoError(t, err)
	for name, url := range map[string]string{
		"normal": tests.GetTestMongoDBDSN(t),
		"ssl":    sslDSN,
	} {
		t.Run(name, func(t *testing.T) {
			testProfiler(t, url)
		})
	}
}

func testProfiler(t *testing.T, url string) {
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
		reports: []*report.Report{},
	}
	prof := New(url, logrus.WithField("component", "profiler-test"), ms, "test-id", truncate.GetMongoDBDefaultMaxQueryLength())
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
			doc[fmt.Sprintf("name_%02d\xff", j)] = fmt.Sprintf("value_%02d\xff", j) // to generate different fingerprints and test UTF8
		}
		dbName := fmt.Sprintf("test_%02d", dbNumber)
		logrus.Tracef("inserting value %d to %s", i, dbName)
		_, err := sess.Database(dbName).Collection("people").InsertOne(context.TODO(), doc)
		assert.NoError(t, err)
		i++
	}
	cursor, err := sess.Database("test_00").Collection("people").Find(context.TODO(), bson.M{"name_00\xff": "value_00\xff"})
	require.NoError(t, err)
	defer cursor.Close(context.TODO())

	<-time.After(aggregator.DefaultInterval * 6) // give it some time to catch all metrics

	err = prof.Stop()
	require.NoError(t, err)

	defer cleanUpDBs(t, sess)

	require.GreaterOrEqual(t, len(ms.reports), 1)

	var findBucket *agentv1.MetricsBucket
	bucketsMap := make(map[string]*agentv1.MetricsBucket)

	for _, r := range ms.reports {
		for _, bucket := range r.Buckets {
			switch bucket.Common.Fingerprint {
			case "db.people.insert(?)":
				key := fmt.Sprintf("%s:%s", bucket.Common.Database, bucket.Common.Fingerprint)
				if b, ok := bucketsMap[key]; ok {
					b.Mongodb.MDocsReturnedCnt += bucket.Mongodb.MDocsReturnedCnt
					b.Mongodb.MResponseLengthCnt += bucket.Mongodb.MResponseLengthCnt
					b.Mongodb.MResponseLengthSum += bucket.Mongodb.MResponseLengthSum
					b.Mongodb.MDocsScannedCnt += bucket.Mongodb.MDocsScannedCnt
				} else {
					bucketsMap[key] = bucket
				}
			case `db.people.find({"name_00\ufffd":"?"})`:
				findBucket = bucket
			default:
				t.Logf("unknown fingerprint: %s", bucket.Common.Fingerprint)
			}
		}
	}

	responseLength := float32(45)

	assert.Equal(t, dbsCount, len(bucketsMap)) // 300 sample docs / 10 = different database names
	var buckets []*agentv1.MetricsBucket
	for _, bucket := range bucketsMap {
		buckets = append(buckets, bucket)
	}
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Common.Database < buckets[j].Common.Database
	})
	for i, bucket := range buckets {
		assert.Equal(t, bucket.Common.Database, fmt.Sprintf("test_%02d", i))
		assert.Equal(t, "db.people.insert(?)", bucket.Common.Fingerprint)
		assert.Equal(t, []string{"people"}, bucket.Common.Tables)
		assert.Equal(t, "test-id", bucket.Common.AgentId)
		assert.Equal(t, inventoryv1.AgentType(9), bucket.Common.AgentType)
		expected := &agentv1.MetricsBucket_MongoDB{
			MDocsReturnedCnt:   docsCount,
			MResponseLengthCnt: docsCount,
			MResponseLengthSum: responseLength * docsCount,
			MResponseLengthMin: responseLength,
			MResponseLengthMax: responseLength,
			MResponseLengthP99: responseLength,
			MDocsScannedCnt:    docsCount,
		}
		// TODO: fix protobuf equality https://jira.percona.com/browse/PMM-6743
		assert.Equalf(t, expected.MDocsReturnedCnt, bucket.Mongodb.MDocsReturnedCnt, "wrong metrics for db %s", bucket.Common.Database)
		assert.Equalf(t, expected.MResponseLengthCnt, bucket.Mongodb.MResponseLengthCnt, "wrong metrics for db %s", bucket.Common.Database)
		assert.Equalf(t, expected.MResponseLengthSum, bucket.Mongodb.MResponseLengthSum, "wrong metrics for db %s", bucket.Common.Database)
		assert.Equalf(t, expected.MResponseLengthMin, bucket.Mongodb.MResponseLengthMin, "wrong metrics for db %s", bucket.Common.Database)
		assert.Equalf(t, expected.MResponseLengthMax, bucket.Mongodb.MResponseLengthMax, "wrong metrics for db %s", bucket.Common.Database)
		assert.Equalf(t, expected.MResponseLengthP99, bucket.Mongodb.MResponseLengthP99, "wrong metrics for db %s", bucket.Common.Database)
		assert.Equalf(t, expected.MDocsScannedCnt, bucket.Mongodb.MDocsScannedCnt, "wrong metrics for db %s", bucket.Common.Database)
	}
	require.NotNil(t, findBucket)
	assert.Equal(t, `db.people.find({"name_00\ufffd":"?"})`, findBucket.Common.Fingerprint)
	assert.Equal(t, docsCount, findBucket.Mongodb.MDocsReturnedSum)
}

func cleanUpDBs(t *testing.T, sess *mongo.Client) {
	dbs, err := sess.ListDatabaseNames(context.TODO(), bson.M{})
	for _, dbname := range dbs {
		if strings.HasPrefix(dbname, "test_") {
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
