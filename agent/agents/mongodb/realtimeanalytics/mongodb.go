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
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	changesBufferSize = 10
	// Timeout for establishing connection to MongoDB.
	mgoConnectTimeout = 5 * time.Second
	// Timeout for MongoDB queries.
	mgoQueryTimeout = 5 * time.Second
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

	// prepare dbAdmin and aggregation pipeline to fetch current operations from MongoDB once
	// to avoid reconstructing them on every collection cycle.
	m.currentOpsPipeline= buildCurrentOpsPipeline()
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
			rtaQueryBucket, err := m.collectCurrentOps(ctx)
			if err != nil {
				m.l.Warnf("CurrentOp collection failed: %v", err)
				continue
			}
			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, RTAQueriesBucket: rtaQueryBucket}
		}
	}
}

// collectCurrentOps runs currentOp command and parses the result into slice of *QueryData.
func (m *MongoDBRTA) collectCurrentOps(ctx context.Context) ([]*rtav1.QueryData, error) {
	cur, err := m.dbAdmin.Aggregate(ctx, m.currentOpsPipeline)
	if err != nil {
		return nil, fmt.Errorf("currentOp not available or permission denied: %w", err)
	}
	defer cur.Close(ctx)

	var results []*rtav1.QueryData
	currTime := timestamppb.New(time.Now())
	for cur.Next(ctx) {
		// check if the context is done to avoid unnecessary processing
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		queryData := parseCurrentOp(cur.Current)
		queryData.ServiceId = m.serviceID
		queryData.ServiceName = m.serviceName
		queryData.CollectTime = currTime
		results = append(results, queryData)
	}
	if err = cur.Err(); err != nil {
		m.l.Warnf("Failed to iterate currentOp cursor: %v", err)
		return nil, err
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

// Helper functions
// createSession creates new MongoDB client and checks connection to MongoDB by pinging it.
func createSession(ctx context.Context, dsn string, agentID string) (*mongo.Client, error) {
	opts, err := clientOptionsForDSN(dsn)
	if err != nil {
		return nil, err
	}

	opts = opts.
		SetDirect(true).
		SetReadPreference(readpref.Nearest()).
		SetTimeout(mgoQueryTimeout).
		SetConnectTimeout(mgoConnectTimeout).
		SetCompressors([]string{"snappy", "zlib", "zstd"}).
		SetAppName(fmt.Sprintf("RTA-mongodb-%s", agentID))

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	if err = client.Ping(ctx, readpref.Nearest()); err != nil {
		return nil, err
	}

	return client, nil
}

// buildCurrentOpsPipeline prepares aggregation pipeline to fetch current operations from MongoDB.
func buildCurrentOpsPipeline() mongo.Pipeline {
	// Prepare aggregation pipeline to fetch current operations.
	// We will use it on every collection cycle, so we prepare it once here to avoid reconstructing it every time.
	// Fetch current operations using aggregation pipeline.
	// Get only active operations for all users.
	selectStage := bson.D{{
		"$currentOp", bson.D{
			{"allUsers", true},
			{"idleConnections", false},
			{"idleCursors", false},
			{"idleSessions", false},
		},
	}}
	// Filter only active operations of type "op" (exclude "command" and other types)
	// to focus on actual queries and reduce amount of data transferred from MongoDB and speed up processing
	matchStage := bson.D{{
		"$match", bson.D{{"active", true}, {"type", "op"}},
	}}
	// Include only fields that we need to reduce amount of data transferred from MongoDB and speed up processing
	projectStage := bson.D{{"$project", bson.D{
		{"cursor", 0},
		{"lsid", 0},
	}}}

	return mongo.Pipeline{selectStage, matchStage, projectStage}
}

// parseCurrentOp parses raw bson document returned by currentOp command into *QueryData.
func parseCurrentOp(raw bson.Raw) *rtav1.QueryData {
	q := &rtav1.QueryData{
		Payload: &rtav1.QueryData_MongoDbPayload{
			MongoDbPayload: &rtav1.QueryMongoDBData{},
		},
	}

	// Generic fields
	if msRunning, ok := raw.Lookup("microsecs_running").Int64OK(); ok {
		q.ExecutionDuration = durationpb.New(time.Duration(1000 * msRunning))
	}
	q.RawQueryJson = raw.String()

	p, _ := q.Payload.(*rtav1.QueryData_MongoDbPayload)
	// MongoDB specific fields
	if opid, ok := raw.Lookup("opid").Int32OK(); ok {
		p.MongoDbPayload.Opid = fmt.Sprintf("%v", opid)
		// there is no separate field for query id in MongoDB, so we will use opid as query id
		q.QueryId = p.MongoDbPayload.Opid
	}

	p.MongoDbPayload.Client, _ = raw.Lookup("client").StringValueOK()
	p.MongoDbPayload.AppName, _ = raw.Lookup("appName").StringValueOK()
	p.MongoDbPayload.WaitingForLock, _ = raw.Lookup("waitingForLock").BooleanOK()
	p.MongoDbPayload.IndexUtilized, _ = raw.Lookup("planSummary").StringValueOK()

	return q
}

// ClientOptionsForDSN applies URI to Client.
func clientOptionsForDSN(dsn string) (*options.ClientOptions, error) {
	clientOptions := options.Client().ApplyURI(dsn)
	if e := clientOptions.Validate(); e != nil {
		return nil, e
	}

	// Workaround for PMM-9320
	// if username or password is set, need to replace it with correctly parsed credentials.
	parsedDsn, err := url.Parse(dsn)
	if err != nil {
		// for non-URI, do nothing (PMM-10265)
		return clientOptions, nil //nolint:nilerr
	}
	username := parsedDsn.User.Username()
	password, _ := parsedDsn.User.Password()
	if username != "" || password != "" {
		clientOptions.Auth.Username = username
		clientOptions.Auth.Password = password
	}

	return clientOptions, nil
}

// check interfaces.
var (
	_ prometheus.Collector = (*MongoDBRTA)(nil)
)
