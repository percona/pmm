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

	"github.com/percona/pmm/agent/agents/mongodb/shared/aggregator"
	"github.com/percona/pmm/agent/agents/mongodb/shared/report"
	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/agent/utils/tests"
	"github.com/percona/pmm/agent/utils/truncate"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

func TestProfiler(t *testing.T) {
	defaultInterval := aggregator.DefaultInterval
	aggregator.DefaultInterval = time.Second
	t.Cleanup(func() {
		aggregator.DefaultInterval = defaultInterval
	})

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

	// Just in case there are old dbs with matching names
	require.NoError(t, cleanUpDBs(t.Context(), t, sess))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.NoError(t, cleanUpDBs(cleanupCtx, t, sess))
	})

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
		_, err = sess.Database(dbName).Collection("test").InsertOne(t.Context(), doc)
		require.NoError(t, err)
		i++
	}
	<-time.After(aggregator.DefaultInterval) // give it some time before starting profiler

	ms := &testWriter{
		t:       t,
		reports: []*report.Report{},
	}
	prof := New(url, logrus.WithField("component", "profiler-test"), ms, "test-id", truncate.GetMongoDBDefaultMaxQueryLength())
	err = prof.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, prof.Stop())
	})
	<-time.After(aggregator.DefaultInterval * 2) // give it some time to start profiler

	i = 0
	for i < dbsCount*int(docsCount) {
		<-ticker.C
		dbNumber := i / int(docsCount)
		fieldsCount := dbNumber + 1
		doc := bson.M{}
		for j := range fieldsCount {
			doc[fmt.Sprintf("name_%02d\xff", j)] = fmt.Sprintf("value_%02d\xff", j) // to generate different fingerprints and test UTF8
		}
		dbName := fmt.Sprintf("test_%02d", dbNumber)
		logrus.Tracef("inserting value %d to %s", i, dbName)
		_, err = sess.Database(dbName).Collection("people").InsertOne(t.Context(), doc)
		require.NoError(t, err)
		i++
	}
	cursor, err := sess.Database("test_00").Collection("people").Find(t.Context(), bson.M{"name_00\xff": "value_00\xff"})
	require.NoError(t, err)
	t.Cleanup(func() {
		cursorCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		assert.NoError(t, cursor.Close(cursorCtx))
	})

	<-time.After(aggregator.DefaultInterval * 6) // give it some time to catch all metrics

	err = prof.Stop()
	require.NoError(t, err)

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
					b.Mongodb.MDocsExaminedCnt += bucket.Mongodb.MDocsExaminedCnt
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

	assert.Len(t, bucketsMap, dbsCount) // 300 sample docs / 10 = different database names
	buckets := make([]*agentv1.MetricsBucket, 0, len(bucketsMap))
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
		}
		// TODO: fix protobuf equality https://jira.percona.com/browse/PMM-6743
		assert.InDeltaf(t, expected.MDocsReturnedCnt, bucket.Mongodb.MDocsReturnedCnt, 0.0001, "wrong metrics MDocsReturnedCnt for db %s", bucket.Common.Database)
		assert.InDeltaf(t, expected.MResponseLengthCnt, bucket.Mongodb.MResponseLengthCnt, 0.0001, "wrong metrics MResponseLengthCnt for db %s", bucket.Common.Database)
		assert.InDeltaf(t, expected.MResponseLengthSum, bucket.Mongodb.MResponseLengthSum, 0.0001, "wrong metrics MResponseLengthSum for db %s", bucket.Common.Database)
		assert.InDeltaf(t, expected.MResponseLengthMin, bucket.Mongodb.MResponseLengthMin, 0.0001, "wrong metrics MResponseLengthMin for db %s", bucket.Common.Database)
		assert.InDeltaf(t, expected.MResponseLengthMax, bucket.Mongodb.MResponseLengthMax, 0.0001, "wrong metrics MResponseLengthMax for db %s", bucket.Common.Database)
		assert.InDeltaf(t, expected.MResponseLengthP99, bucket.Mongodb.MResponseLengthP99, 0.0001, "wrong metrics MResponseLengthP99 for db %s", bucket.Common.Database)
		assert.InDeltaf(t, expected.MDocsExaminedCnt, bucket.Mongodb.MDocsExaminedCnt, 0.0001, "wrong metrics MDocsExaminedCnt for db %s", bucket.Common.Database)
	}
	require.NotNil(t, findBucket)
	assert.Equal(t, `db.people.find({"name_00\ufffd":"?"})`, findBucket.Common.Fingerprint)
	assert.InDelta(t, docsCount, findBucket.Mongodb.MDocsReturnedSum, 0.0001)
}

func cleanUpDBs(ctx context.Context, t *testing.T, sess *mongo.Client) error {
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

type testWriter struct {
	t       *testing.T
	reports []*report.Report
}

func (tw *testWriter) Write(actual *report.Report) error {
	require.NotNil(tw.t, actual)
	tw.reports = append(tw.reports, actual)
	return nil
}
