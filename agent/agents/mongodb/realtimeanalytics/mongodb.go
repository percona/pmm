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

// Package realtimeanalytics runs built-in Real-Time Analytics Agent for MongoDB.
package realtimeanalytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	rtaParser "github.com/percona/pmm/agent/agents/mongodb/realtimeanalytics/parser"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	changesBufferSize = 10
)

// MongoDBRTA extracts Real-Time Analytics data (currently running DB queries) from MongoDB.
type MongoDBRTA struct {
	agentID     string
	serviceID   string
	serviceName string
	l           *logrus.Entry

	// Channel to obtain data from this agent.
	changes chan agents.Change

	// DSN to connect to MongoDB.
	mongoDSN string
	// collectInterval is how often to collect data from MongoDB.
	collectInterval time.Duration
	// currentOpsPipeline is the aggregation pipeline used to fetch current operations from MongoDB.
	// We can keep it as a field to avoid reconstructing it on every collection cycle.
	currentOpsPipeline mongo.Pipeline
	// dbAdmin is the admin database of MongoDB, we can keep it as a field to avoid getting it on every collection cycle.
	dbAdmin *mongo.Database
}

// Params represent Agent parameters.
type Params struct {
	AgentID         string
	DSN             string        // DSN to connect to MongoDB.
	ServiceID       string        // ServiceID shall be set in RTA queries to link them to the service.
	ServiceName     string        // ServiceName shall be set in RTA queries to link them to the service.
	CollectInterval time.Duration // CollectInterval is how often to collect data from MongoDB.
}

// New creates new MongoDBRTA service.
func New(params *Params, l *logrus.Entry) (*MongoDBRTA, error) {
	// if params.DSN is incorrect we should exit immediately as this is not gonna correct itself
	_, err := connstring.Parse(params.DSN)
	if err != nil {
		return nil, err
	}

	return &MongoDBRTA{
		agentID:         params.AgentID,
		serviceID:       params.ServiceID,
		serviceName:     params.ServiceName,
		mongoDSN:        params.DSN,
		collectInterval: params.CollectInterval,
		l:               l,
		changes:         make(chan agents.Change, changesBufferSize),
		// prepare aggregation pipeline to fetch current operations from MongoDB once
		// to avoid reconstructing it on every collection cycle.
		currentOpsPipeline: buildCurrentOpsPipeline(),
	}, nil
}

// Run extracts currently running DB queries from MongoDB
// and sends it to the channel until ctx is canceled.
func (m *MongoDBRTA) Run(ctx context.Context) {
	m.l.Info("Starting MongoDB RTA agent")

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	defer func() {
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}

		close(m.changes)
	}()

	// create connection to MongoDB
	client, err := createSession(ctx, m.mongoDSN, m.agentID)
	if err != nil {
		m.l.Errorf("Can't run Real-Time Analytics agent, reason: %v", err)

		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}

		return
	}

	defer func() {
		_ = client.Disconnect(ctx)
	}()

	// prepare dbAdmin to fetch current operations from MongoDB once
	// to avoid reconstructing it on every collection cycle.
	m.dbAdmin = client.Database("admin")

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
	// fetch RTA data periodically
	ticker := time.NewTicker(m.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.l.Info("Stopping MongoDB RTA agent")

			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			// m.changes channel will be closed in defer, so we don't need to close it here, just exit the function
			return
		case <-ticker.C:
			// We run collection in a separate goroutine to avoid blocking the main loop
			// and allow timely execution of next ticks in case collection/parsing takes longer
			// than the collect interval.
			go func(curCtx context.Context) {
				rtaQueryBucket, err := m.collectCurrentOps(curCtx)
				if err != nil {
					m.l.Warnf("CurrentOp collection failed: %v", err)
					return
				}

				select {
				case <-curCtx.Done():
					// If context is done, we don't send anything to the channel.
					return
				default:
					change := agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
					if len(rtaQueryBucket) != 0 {
						// If we have data, send it to the channel.
						// If not, send only status without data to avoid triggering
						// unnecessary processing in the receiver.
						change.RTAQueriesBucket = rtaQueryBucket
					}

					m.changes <- change
				}
			}(ctx)
		}
	}
}

// collectCurrentOps runs currentOp command and parses the result into slice of *QueryData.
func (m *MongoDBRTA) collectCurrentOps(ctx context.Context) ([]*rtav1.QueryData, error) {
	cur, err := m.dbAdmin.Aggregate(ctx, m.currentOpsPipeline)
	if err != nil {
		return nil, fmt.Errorf("currentOp not available or permission denied: %w", err)
	}

	defer func() {
		_ = cur.Close(ctx)
	}()

	resultLen := cur.RemainingBatchLength()
	if resultLen == 0 {
		// If there are no current operations, we can return early with an empty slice
		// to avoid unnecessary processing.
		return nil, nil
	}

	results := make([]*rtav1.QueryData, 0, resultLen)
	currTime := timestamppb.New(time.Now())

	// Parallel parsing of currentOp results using WaitGroup and channel to store
	// results from goroutines safely to speed up processing of large number of current operations.
	wg := sync.WaitGroup{}

	// Buffered channel to store results from goroutines(parsers)
	// and avoid blocking when sending results to the channel.
	resultsChan := make(chan *rtav1.QueryData, 10) //nolint:mnd

	for cur.Next(ctx) {
		// check if the context is done to avoid unnecessary processing
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)

		go func(curCopy mongo.Cursor, ch chan<- *rtav1.QueryData) {
			defer wg.Done()

			queryData := rtaParser.ParseCurrentOp(curCopy.Current)
			if queryData == nil {
				// If parsing failed, we skip this operation and don't include it in the results.
				return
			}

			queryData.ServiceId = m.serviceID
			queryData.ServiceName = m.serviceName

			queryData.QueryCollectTime = currTime
			ch <- queryData
		}(*cur, resultsChan) // We create a copy of the cursor to avoid unwanted overrides.
	}
	// Wait for all parsing goroutines to finish and
	// close the results channel to signal that no more results will be sent.
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	err = cur.Err()
	if err != nil {
		m.l.Warnf("Failed to iterate currentOp cursor: %v", err)
		return nil, err
	}

	for q := range resultsChan {
		results = append(results, q)
	}

	return results, nil
}

// Changes returns channel that should be read until it is closed.
func (m *MongoDBRTA) Changes() <-chan agents.Change {
	return m.changes
}

// Describe implements prometheus.Collector.
func (m *MongoDBRTA) Describe(_ chan<- *prometheus.Desc) {
	// This method is needed to satisfy interface.
}

// Collect implement prometheus.Collector.
func (m *MongoDBRTA) Collect(_ chan<- prometheus.Metric) {
	// This method is needed to satisfy interface.
}

// Helper functions.

// buildCurrentOpsPipeline prepares aggregation pipeline to fetch current operations from MongoDB.
func buildCurrentOpsPipeline() mongo.Pipeline {
	// Prepare aggregation pipeline to fetch current operations.
	// We will use it on every collection cycle, so we prepare it once here to avoid reconstructing it every time.
	// Fetch current operations using aggregation pipeline.
	// Get only active operations for all users.
	selectStage := bson.D{{
		Key: "$currentOp", Value: bson.D{
			{Key: "allUsers", Value: true},
			{Key: "idleConnections", Value: false},
			{Key: "idleCursors", Value: false},
			{Key: "idleSessions", Value: false},
		},
	}}
	// Filter only active operations of type "op" (exclude "command" and other types)
	// to focus on actual queries and reduce amount of data transferred from MongoDB and speed up processing

	/*
		The matchStage below equal to the following aggregation pipeline in MongoDB shell,
		which filters out operations from RTA agent itself and internal MongoDB tools, and gets only active operations/commands:
		db.aggregate([
		    { $currentOp : { allUsers: true, idleSessions: false,  idleCursors:false, idleConnections:false} },
		    { $match : {
		        $and: [
		            {"appName": {$not: {$regex: "^(rta-mongodb-.*$)"}}},
		            { "desc": {$nin: ["Checkpointer", "JournalFlusher"]}},
		            { active: true}
		            ],
		        }
		        }
		    ]);
	*/
	matchStage := bson.D{{
		Key: "$match", Value: bson.D{
			{
				Key: "$and", Value: bson.A{
					// Get operations/commands that are active.
					bson.D{{Key: "active", Value: true}},
					// Exclude operations from internal MongoDB tools.
					bson.D{{Key: "desc", Value: bson.D{{Key: "$nin", Value: bson.A{"Checkpointer", "JournalFlusher"}}}}},
					// Exclude operations from RTA agent itself.
					bson.D{{Key: "appName", Value: bson.D{{Key: "$not", Value: bson.D{{Key: "$regex", Value: "^(rta-mongodb-.*$)"}}}}}},
				},
			},
		},
	}}

	return mongo.Pipeline{selectStage, matchStage}
}

// check interfaces.
var (
	_ prometheus.Collector = (*MongoDBRTA)(nil)
)
