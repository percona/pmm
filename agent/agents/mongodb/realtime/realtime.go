// Package realtime provides a MongoDB real-time query collector agent for PMM.
package realtime

import (
	context "context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)


import (
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)


// parseCurrentOp extracts fields from a currentOp document and returns proto struct.
func parseCurrentOp(raw bson.M) *agentv1.MongoDBRealtimeQueryData {
	d := &agentv1.MongoDBRealtimeQueryData{}
	if v, ok := raw["opid"]; ok {
		d.Opid = fmt.Sprintf("%v", v)
	}
	if v, ok := raw["secs_running"]; ok {
		d.SecsRunning, _ = toInt64(v)
		d.CurrentExecutionTime = float64(d.SecsRunning)
	}
	if v, ok := raw["client"]; ok {
		d.Client, _ = v.(string)
	}
	if v, ok := raw["waitingForLock"]; ok {
		d.WaitingForLock, _ = v.(bool)
	}
	if v, ok := raw["planSummary"]; ok {
		plan, _ := v.(string)
		if strings.Contains(plan, "IXSCAN") {
			d.IndexUtilized = true
		}
	}
	if v, ok := raw["query"].(bson.M); ok {
		q, _ := json.Marshal(v)
		d.QueryText = string(q)
	}
	if v, ok := raw["command"].(bson.M); ok && d.QueryText == "" {
		q, _ := json.Marshal(v)
		d.QueryText = string(q)
	}
	if v, ok := raw["ns"]; ok {
		d.QueryId, _ = v.(string)
	}
	if v, ok := raw["state"]; ok {
		d.State, _ = v.(string)
	}
	if v, ok := raw["numYields"]; ok {
		d.RowsExamined, _ = toInt64(v)
	}
	if v, ok := raw["nreturned"]; ok {
		d.RowsSent, _ = toInt64(v)
	}
	d.Timestamp = timestamppb.Now()
	if raw != nil {
		jsonRaw, _ := json.Marshal(raw)
		d.RawQueryJson = string(jsonRaw)
	}
	return d
}

// Config holds agent configuration.
type Config struct {
	URI             string
	PollingInterval time.Duration
}


// Agent collects real-time queries from MongoDB.
type Agent struct {
	cfg    Config
	logger *logrus.Entry
}

// New creates a new MongoDB real-time agent.
func New(cfg Config, logger *logrus.Entry) *Agent {
	return &Agent{cfg: cfg, logger: logger.WithField("agent", "mongodb-realtime")}
}

// Run starts the agent main loop.
func (a *Agent) Run(ctx context.Context, out chan<- *agentv1.MongoDBRealtimeQueryData) {
	defer func() {
		if r := recover(); r != nil {
			a.logger.Errorf("panic recovered: %v", r)
		}
	}()

       client, err := mongo.Connect(ctx, options.Client().ApplyURI(a.cfg.URI))
       if err != nil {
	       a.logger.Errorf("failed to connect to MongoDB: %v", err)
	       return
       }
       defer func() {
	       _ = client.Disconnect(ctx)
       }()

       if err := client.Ping(ctx, readpref.Primary()); err != nil {
	       a.logger.Errorf("MongoDB ping failed: %v", err)
	       return
       }

       ticker := time.NewTicker(a.cfg.PollingInterval)
       defer ticker.Stop()

       for {
	       select {
	       case <-ctx.Done():
		       a.logger.Info("context cancelled, stopping agent")
		       return
	       case <-ticker.C:
		       ops, err := a.collectCurrentOps(ctx, client)
		       if err != nil {
			       a.logger.Warnf("currentOp collection failed: %v", err)
			       continue
		       }
		       for _, op := range ops {
			       select {
			       case out <- op:
			       case <-ctx.Done():
				       return
			       }
		       }
	       }
       }
}

// collectCurrentOps executes db.currentOp() and parses results.
func (a *Agent) collectCurrentOps(ctx context.Context, client *mongo.Client) ([]*agentv1.MongoDBRealtimeQueryData, error) {
       admin := client.Database("admin")
       cmd := bson.D{{Key: "currentOp", Value: 1}}
       cur, err := admin.RunCommandCursor(ctx, cmd)
       if err != nil {
	       return nil, fmt.Errorf("currentOp not available or permission denied: %w", err)
       }
       defer cur.Close(ctx)

       var results []*agentv1.MongoDBRealtimeQueryData
       for cur.Next(ctx) {
	       var raw bson.M
	       if err := cur.Decode(&raw); err != nil {
		       a.logger.Warnf("failed to decode currentOp result: %v", err)
		       continue
	       }
	       data := parseCurrentOp(raw)
	       results = append(results, data)
       }
       return results, nil
}

// ...existing code...

func toInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case int32:
		return int64(t), true
	case int64:
		return t, true
	case float64:
		return int64(t), true
	case float32:
		return int64(t), true
	case uint32:
		return int64(t), true
	case uint64:
		return int64(t), true
	case int:
		return int64(t), true
	}
	return 0, false
}
