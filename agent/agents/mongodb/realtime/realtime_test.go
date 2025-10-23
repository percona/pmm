package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	realtimev1 "github.com/percona/pmm/api/realtime/v1"
)

const (
	testMongoURI = "mongodb://root:root-password@localhost:27017/admin"
	testTimeout  = 30 * time.Second
)

// TestRTA_CaptureSlowQuery tests the RTA agent's ability to capture a running slow query
func TestRTA_CaptureSlowQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.WithField("component", "rta-test")

	params := &Params{
		DSN:                     testMongoURI,
		AgentID:                 "test-rta-agent",
		ServiceID:               "test-service",
		ServiceName:             "test-mongodb",
		NodeID:                  "test-node",
		NodeName:                "test-node",
		Labels:                  map[string]string{"env": "test"},
		CollectionInterval:      100 * time.Millisecond, // Fast collection to catch queries
		DisableQueryText:        false,
		MaxQueriesPerCollection: 50,
	}

	agent, err := New(params, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	err = agent.connect(ctx)
	require.NoError(t, err)
	defer agent.disconnect()

	// Create test client for running queries
	testClient, err := mongo.Connect(ctx, options.Client().ApplyURI(testMongoURI))
	require.NoError(t, err)
	defer testClient.Disconnect(ctx)

	// Setup test data
	setupTestData(t, testClient)

	t.Log("🔍 Starting slow query capture test")

	// Start a slow aggregation query that should be captured
	queryCtx, queryCancel := context.WithTimeout(ctx, 10*time.Second)
	defer queryCancel()

	queryDone := make(chan error, 1)
	go func() {
		defer close(queryDone)

		db := testClient.Database("rtaTestDB")

		// Run a slow aggregation with JavaScript function to ensure it takes time
		pipeline := []bson.M{
			{"$match": bson.M{"value": bson.M{"$exists": true}}},
			{"$addFields": bson.M{
				"slow_calculation": bson.M{
					"$function": bson.M{
						"body": `function(value) { 
							// Simulate slow processing
							var start = Date.now(); 
							while(Date.now() - start < 500) {} // 500ms delay per document
							return value * 2; 
						}`,
						"args": []interface{}{"$value"},
						"lang": "js",
					},
				},
			}},
			{"$group": bson.M{
				"_id":   "$category",
				"total": bson.M{"$sum": "$slow_calculation"},
				"count": bson.M{"$sum": 1},
			}},
			{"$sort": bson.M{"total": -1}},
		}

		t.Log("⚡ Starting slow aggregation query...")
		cursor, err := db.Collection("testData").Aggregate(queryCtx, pipeline)
		if err != nil {
			queryDone <- err
			return
		}
		defer cursor.Close(queryCtx)

		// Process results to keep the query running
		var results []bson.M
		err = cursor.All(queryCtx, &results)
		queryDone <- err
	}()

	// Give the query a moment to start
	time.Sleep(200 * time.Millisecond)

	// Try to capture the running query
	var capturedQueries []*realtimev1.RealTimeQueryData
	maxAttempts := 30 // 3 seconds of attempts
	foundRunning := false

	t.Log("🎯 Attempting to capture running query...")
	for attempt := 0; attempt < maxAttempts; attempt++ {
		ops, err := agent.getCurrentOperations(ctx)
		require.NoError(t, err)

		if len(ops) > 0 {
			t.Logf("📊 Attempt %d: Found %d operations", attempt+1, len(ops))

			for _, op := range ops {
				if op.State == realtimev1.QueryState_RUNNING {
					capturedQueries = append(capturedQueries, op)
					foundRunning = true
					t.Logf("✅ CAPTURED RUNNING QUERY!")
					t.Logf("   Database: %s", op.Database)
					t.Logf("   Operation: %s", op.Mongodb.OperationType)
					t.Logf("   Duration: %.2fs", op.CurrentExecutionTime)
					t.Logf("   Fingerprint: %s", op.Fingerprint)
					if op.QueryText != "" {
						t.Logf("   Query Text: %s", op.QueryText)
					}
				}
			}
		} else {
			t.Logf("⏳ Attempt %d: No operations found", attempt+1)
		}

		if foundRunning {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Wait for query to complete
	select {
	case err := <-queryDone:
		if err != nil {
			t.Logf("❌ Query completed with error: %v", err)
		} else {
			t.Log("✅ Query completed successfully")
		}
	case <-time.After(5 * time.Second):
		t.Log("⏰ Query timed out")
		queryCancel()
	}

	// Report final results
	t.Log("📈 TEST RESULTS:")
	t.Logf("   Found running queries: %v", foundRunning)
	t.Logf("   Total captured operations: %d", len(capturedQueries))
	t.Logf("   Capture attempts made: %d", maxAttempts)

	// Assert that we actually captured a slow running query
	require.True(t, foundRunning, "Expected to capture at least one running query, but none were found")
	require.Greater(t, len(capturedQueries), 0, "Expected to capture at least one operation")

	// Validate the captured query details
	capturedQuery := capturedQueries[0]
	require.Equal(t, "rtaTestDB", capturedQuery.Database, "Expected database to be rtaTestDB")
	require.Equal(t, realtimev1.QueryState_RUNNING, capturedQuery.State, "Expected query state to be RUNNING")
	require.NotEmpty(t, capturedQuery.Fingerprint, "Expected fingerprint to be present")

	t.Log("🎉 SUCCESS: RTA agent successfully captured a slow running query!")
}

// setupTestData creates test data for the slow query
func setupTestData(t *testing.T, client *mongo.Client) {
	ctx := context.Background()

	db := client.Database("rtaTestDB")
	collection := db.Collection("testData")

	// Drop existing data
	_ = collection.Drop(ctx)

	// Insert test documents (fewer documents for faster setup)
	docs := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		docs[i] = bson.M{
			"_id":      i,
			"value":    i + 1,
			"category": i % 5, // 5 categories
		}
	}

	_, err := collection.InsertMany(ctx, docs)
	require.NoError(t, err)

	t.Log("📋 Test data setup complete - 100 documents inserted")
}
