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

// Package realtime provides real-time MongoDB query analytics.
package realtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/agents/mongodb/shared/fingerprinter"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	realtimev1 "github.com/percona/pmm/api/realtime/v1"
)

// MongoDB extracts real-time performance data from MongoDB currentOp.
type MongoDB struct {
	agentID     string
	serviceID   string
	serviceName string
	nodeID      string
	nodeName    string
	labels      map[string]string
	l           *logrus.Entry
	changes     chan agents.Change
	client      *mongo.Client

	mongoDSN                string
	collectionInterval      time.Duration
	disableQueryText        bool
	maxQueriesPerCollection int32

	// Internal state
	mu      sync.RWMutex
	running bool

	// Fingerprinter for consistent query fingerprinting
	fingerprinter *fingerprinter.ProfilerFingerprinter

	// Metrics
	queriesCollected prometheus.Counter
	collectDuration  prometheus.Histogram
}

// Params represent Agent parameters.
type Params struct {
	DSN                     string
	AgentID                 string
	ServiceID               string
	ServiceName             string
	NodeID                  string
	NodeName                string
	Labels                  map[string]string
	CollectionInterval      time.Duration
	DisableQueryText        bool
	MaxQueriesPerCollection int32
}

// RealTimeAnalyticsCollector defines the interface for sending real-time data.
type RealTimeAnalyticsCollector interface {
	SendRealTimeData(data []*realtimev1.RealTimeQueryData) error
}

// New creates new MongoDB real-time analytics agent.
func New(params *Params, l *logrus.Entry) (*MongoDB, error) {
	// Validate DSN
	_, err := connstring.Parse(params.DSN)
	if err != nil {
		return nil, fmt.Errorf("invalid MongoDB DSN: %w", err)
	}

	// Set defaults
	collectionInterval := params.CollectionInterval
	if collectionInterval <= 0 {
		collectionInterval = time.Second
	}

	maxQueries := params.MaxQueriesPerCollection
	if maxQueries <= 0 {
		maxQueries = 100
	}

	return &MongoDB{
		agentID:                 params.AgentID,
		serviceID:               params.ServiceID,
		serviceName:             params.ServiceName,
		nodeID:                  params.NodeID,
		nodeName:                params.NodeName,
		labels:                  params.Labels,
		mongoDSN:                params.DSN,
		collectionInterval:      collectionInterval,
		disableQueryText:        params.DisableQueryText,
		maxQueriesPerCollection: maxQueries,
		l:                       l,
		changes:                 make(chan agents.Change, 10),

		// Initialize fingerprinter
		fingerprinter: fingerprinter.NewFingerprinter(fingerprinter.DefaultKeyFilters()),

		// Initialize metrics
		queriesCollected: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "pmm_agent",
			Subsystem: "mongodb_realtime",
			Name:      "queries_collected_total",
			Help:      "Total number of queries collected by MongoDB real-time analytics agent.",
		}),
		collectDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "pmm_agent",
			Subsystem: "mongodb_realtime",
			Name:      "collect_duration_seconds",
			Help:      "Time spent collecting MongoDB real-time data.",
		}),
	}, nil
}

// Run extracts real-time performance data and sends it until ctx is canceled.
func (m *MongoDB) Run(ctx context.Context) {
	defer func() {
		m.disconnect()
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(m.changes)
	}()

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	if err := m.connect(ctx); err != nil {
		m.l.Errorf("Failed to connect to MongoDB: %v", err)
		m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
		return
	}

	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}

	ticker := time.NewTicker(m.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.l.Info("Real-time analytics agent stopping")
			m.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING}
			return
		case <-ticker.C:
			if err := m.collectAndSend(ctx); err != nil {
				m.l.Errorf("Failed to collect real-time data: %v", err)
			}
		}
	}
}

// connect establishes connection to MongoDB.
func (m *MongoDB) connect(ctx context.Context) error {
	clientOptions := options.Client().ApplyURI(m.mongoDSN)

	// Set connection timeout
	connectTimeout := 10 * time.Second
	clientOptions.SetConnectTimeout(connectTimeout)
	clientOptions.SetServerSelectionTimeout(connectTimeout)

	var err error
	m.client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := m.client.Ping(ctx, nil); err != nil {
		m.client.Disconnect(context.Background()) //nolint:errcheck
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.l.Info("Successfully connected to MongoDB for real-time analytics")
	return nil
}

// disconnect closes MongoDB connection.
func (m *MongoDB) disconnect() {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := m.client.Disconnect(ctx); err != nil {
			m.l.Errorf("Failed to disconnect from MongoDB: %v", err)
		}
		m.client = nil
	}
}

// collectAndSend collects current operations and sends them to the collector.
func (m *MongoDB) collectAndSend(ctx context.Context) error {
	start := time.Now()
	defer func() {
		m.collectDuration.Observe(time.Since(start).Seconds())
	}()

	currentOps, err := m.getCurrentOperations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current operations: %w", err)
	}

	if len(currentOps) == 0 {
		m.l.Debug("No current operations found")
		return nil
	}

	m.l.Debugf("Collected %d current operations", len(currentOps))
	m.queriesCollected.Add(float64(len(currentOps)))

	// Send collected real-time queries through the changes channel
	runningQueries := make([]*realtimev1.RealTimeQueryData, 0)
	for _, queryData := range currentOps {
		runningQueries = append(runningQueries, queryData)

		m.l.Debugf("RTA: Running query - Database: %s, Operation: %s, Duration: %.3fs (%.0fμs), Fingerprint: %s",
			queryData.Database,
			queryData.Mongodb.OperationType,
			queryData.CurrentExecutionTime,
			queryData.CurrentExecutionTime*1000000, // Show microseconds
			queryData.Fingerprint)

		if !m.disableQueryText && queryData.QueryText != "" {
			m.l.Debugf("RTA: Query text: %s", queryData.QueryText)
		}
	}

	// Send real-time data if we have running queries
	if len(runningQueries) > 0 {
		m.changes <- agents.Change{
			RealTimeQueries: runningQueries,
		}
	}

	return nil
}

// currentOp represents the structure of a MongoDB currentOp document
type currentOp struct {
	Opid             int64  `bson:"opid"`
	Active           bool   `bson:"active"`
	Ns               string `bson:"ns"`
	Op               string `bson:"op"`
	MicrosecsRunning int64  `bson:"microsecs_running"` // MongoDB uses NumberLong for microseconds
	SecsRunning      int64  `bson:"secs_running"`      // MongoDB uses NumberLong for seconds
	Client           string `bson:"client"`
	Command          bson.M `bson:"command"`
	WaitingForLock   bool   `bson:"waitingForLock"`
	PlanSummary      string `bson:"planSummary"`
}

// getCurrentOperations retrieves current operations from MongoDB using currentOp command.
func (m *MongoDB) getCurrentOperations(ctx context.Context) ([]*realtimev1.RealTimeQueryData, error) {
	// Set timeout for the operation
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Run currentOp command
	db := m.client.Database("admin")

	// Execute currentOp aggregation with proper filtering
	// Note: $currentOp doesn't accept active/microsecs_running in options, we filter after
	pipeline := []bson.M{
		{"$currentOp": bson.M{"allUsers": true, "idleConnections": true}},
		{"$match": bson.M{
			"microsecs_running": bson.M{"$gte": 1000}, // 1ms threshold to catch operations
		}},
	}

	// Run aggregation with proper options
	opts := options.Aggregate().SetAllowDiskUse(true)
	cursor, err := db.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to execute currentOp aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	results := make([]*realtimev1.RealTimeQueryData, 0)
	timestamp := timestamppb.Now()

	for cursor.Next(ctx) {
		var op currentOp
		if err := cursor.Decode(&op); err != nil {
			m.l.Errorf("Failed to decode currentOp result: %v", err)
			continue
		}

		// Convert currentOp data to systemProfile format for fingerprinting
		var sysProfile proto.SystemProfile
		m.convertCurrentOpToSystemProfile(&op, &sysProfile)

		// Generate fingerprint using the converted systemProfile
		fingerprint, err := m.fingerprinter.Fingerprint(sysProfile)
		if err != nil {
			m.l.Warnf("Failed to fingerprint system.profile result: %v", err)
		}

		queryData := m.convertCurrentOpToRealTimeData(&op, cursor.Current, fingerprint.Fingerprint, timestamp)
		if queryData != nil {
			results = append(results, queryData)
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while reading currentOp results: %w", err)
	}

	return results, nil
}

// convertCurrentOpToRealTimeData converts a currentOp struct to RealTimeQueryData.
func (m *MongoDB) convertCurrentOpToRealTimeData(op *currentOp, currentOpRaw bson.Raw, fingerprint string, timestamp *timestamppb.Timestamp) *realtimev1.RealTimeQueryData {
	// Convert timing from MongoDB fields - prefer microsecs_running for precision
	var secsRunning float64
	if op.MicrosecsRunning > 0 {
		secsRunning = float64(op.MicrosecsRunning) / 1000000.0 // Convert microseconds to seconds
	} else if op.SecsRunning > 0 {
		secsRunning = float64(op.SecsRunning) // Use seconds directly as fallback
	}

	// Extract database name from namespace
	var database string
	if op.Ns != "" {
		if dotIndex := len(op.Ns); dotIndex > 0 {
			for i, char := range op.Ns {
				if char == '.' {
					dotIndex = i
					break
				}
			}
			if dotIndex > 0 {
				database = op.Ns[:dotIndex]
			}
		}
	}

	// Generate query text from command
	var queryText string
	if op.Command != nil && !m.disableQueryText {
		if commandBytes, err := bson.MarshalExtJSON(op.Command, false, false); err == nil {
			queryText = string(commandBytes)
		}
	}

	// Use fallback fingerprint if needed
	if fingerprint == "" {
		m.l.Debugf("Fingerprinter failed: using fallback")
		fingerprint = fmt.Sprintf("%s.%s", op.Op, database)
	}

	// Determine query state
	var state realtimev1.QueryState
	if op.Active {
		if op.WaitingForLock {
			state = realtimev1.QueryState_WAITING
		} else {
			state = realtimev1.QueryState_RUNNING
		}
	} else {
		state = realtimev1.QueryState_FINISHED
	}

	// Convert currentOp raw BSON to JSON string
	var currentOpJSON string
	if currentOpRawBytes, err := bson.MarshalExtJSON(currentOpRaw, false, false); err == nil {
		currentOpJSON = string(currentOpRawBytes)
	} else {
		m.l.Debugf("Failed to convert currentOp to JSON: %v", err)
	}

	// Extract MongoDB-specific fields
	mongoFields := &realtimev1.MongoDBFields{
		Opid:          op.Opid,
		SecsRunning:   secsRunning,
		OperationType: op.Op,
		Namespace:     op.Ns,
		Blocking:      op.WaitingForLock,
		CurrentOpRaw:  currentOpJSON,
	}

	return &realtimev1.RealTimeQueryData{
		QueryId:              fmt.Sprintf("%d", op.Opid),
		QueryText:            queryText,
		Fingerprint:          fingerprint,
		Database:             database,
		ClientHost:           op.Client,
		Timestamp:            timestamp,
		State:                state,
		CurrentExecutionTime: secsRunning,
		Mongodb:              mongoFields,
		// Service metadata
		ServiceId:   m.serviceID,
		ServiceName: m.serviceName,
		NodeId:      m.nodeID,
		NodeName:    m.nodeName,
		Labels:      m.labels,
	}
}

// convertCurrentOpToSystemProfile converts currentOp struct to systemProfile format for fingerprinting
// Only populates the fields that the fingerprinter actually uses: Ns, Op, and Command
func (m *MongoDB) convertCurrentOpToSystemProfile(op *currentOp, sysProfile *proto.SystemProfile) {
	// Set fields required by fingerprinter
	sysProfile.Ns = op.Ns
	sysProfile.Op = op.Op

	// Convert command from bson.M to bson.D (fingerprinter calls .Map() on it anyway)
	if op.Command != nil {
		commandD := make(bson.D, 0, len(op.Command))
		for key, value := range op.Command {
			commandD = append(commandD, bson.E{Key: key, Value: value})
		}
		sysProfile.Command = commandD
	}
}

// Changes returns channel that should be read until it is closed.
func (m *MongoDB) Changes() <-chan agents.Change {
	return m.changes
}

// Describe implements prometheus.Collector.
func (m *MongoDB) Describe(ch chan<- *prometheus.Desc) {
	m.queriesCollected.Describe(ch)
	m.collectDuration.Describe(ch)
}

// Collect implements prometheus.Collector.
func (m *MongoDB) Collect(ch chan<- prometheus.Metric) {
	m.queriesCollected.Collect(ch)
	m.collectDuration.Collect(ch)
}
