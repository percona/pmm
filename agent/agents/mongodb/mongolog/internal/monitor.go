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

package mongolog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/percona/pmm/agent/utils/filereader"
)

const (
	slowQuery       = "Slow query"
	authQuery       = "Successfully authenticated"
	disconnectQuery = "Connection ended"
)

// NewMonitor creates new monitor.
func NewMonitor(logPath string, reader *filereader.ContinuousFileReader, logger *logrus.Entry) *Monitor {
	return &Monitor{
		logPath: logPath,
		reader:  reader,
		logger:  logger,
	}
}

// Monitor represents mongolog aggregator and helpers.
type Monitor struct {
	// dependencies
	logPath string
	reader  *filereader.ContinuousFileReader
	logger  *logrus.Entry

	// state
	m       sync.Mutex
	running bool
}

// Start starts monitor to collect and parse data.
func (m *Monitor) Start(ctx context.Context, docsChan chan proto.SystemProfile, doneChan <-chan struct{}, wg *sync.WaitGroup) {
	m.m.Lock()
	defer m.m.Unlock()

	if m.running {
		return
	}

	go func() {
		readFile(ctx, m.reader, docsChan, doneChan, wg, m.logger)
		m.logger.Debugln("done reading the log file")

		m.m.Lock()
		defer m.m.Unlock()
		m.running = false
	}()

	m.running = true
}

// row is a helper structure to unmarshall Monglog row to system.Profile metrics.
type row struct {
	T struct {
		Date time.Time `json:"$date"`
	} `json:"t"`
	Msg  string `json:"msg"`
	Attr json.RawMessage
}

type systemProfile struct {
	proto.SystemProfile
	Command bson.M `json:"command"`
	Type    string `json:"type"`
	Remote  string `json:"remote"`
}

type auth struct {
	// Connect
	Client string `json:"client"`
	User   string `json:"user"`
	// Disconnect
	Remote string `json:"remote"`
}

func fixOpAndNs(profile *systemProfile) {
	if profile.Type != "" {
		profile.Op = profile.Type
		return
	}

	switch profile.Protocol {
	case "op_msg":
		// Check command keys to identify operation type
		for k, v := range profile.Command {
			switch k {
			case "insert":
				profile.Op = "insert"
			case "find":
				profile.Op = "query"
			case "update":
				profile.Op = "update"
			case "delete":
				profile.Op = "remove"
			case "explain", "endSessions", "killCursors":
				profile.Op = "command"
			}

			profile.Ns = fmt.Sprintf("%s.%s", profile.Ns, v)
			return
		}
		// Default for op_msg without a known command key
		profile.Op = "command"
		// not known ns?
		return

		// If protocol is op_query
	case "op_query":
		profile.Op = "query"
		// not known ns?
		return
	default:
		profile.Op = profile.Protocol
		// not known ns?
	}
}

// readFile continuously read new lines from file, until it is canceled or considered as done.
func readFile(ctx context.Context, reader *filereader.ContinuousFileReader, docsChan chan proto.SystemProfile,
	doneChan <-chan struct{}, wg *sync.WaitGroup, logger *logrus.Entry,
) {
	defer wg.Done()
	logger.Debugln("reader started")

	connections := map[string]string{}
	for {
		select {
		case <-ctx.Done():
			logger.Debugln("context done")
			return
		case <-doneChan:
			logger.Debugln("reader done")
			return
		default:
			logger.Debugln("reading a line")
			line, err := reader.NextLine()
			if err != nil {
				logger.Error(err)
				return
			}
			logger.Debugf("readed line: %s", line)

			var l row
			if line == "" || !json.Valid([]byte(line)) {
				continue
			}
			err = json.Unmarshal([]byte(line), &l) //nolint:musttag
			if err != nil {
				logger.Error(err)
				continue
			}

			switch l.Msg {
			case slowQuery:
				sendQuery(l, logger, docsChan, connections)
			case authQuery:
				if connection, ok := getConnection(l.Attr, logger); ok {
					connections[connection.Client] = connection.User
				}
			case disconnectQuery:
				if connection, ok := getConnection(l.Attr, logger); ok {
					delete(connections, connection.Remote)
				}
			}
		}
	}
}

func getConnection(attr json.RawMessage, logger *logrus.Entry) (auth, bool) {
	var connection auth
	err := json.Unmarshal(attr, &connection)
	if err != nil {
		logger.Debugln("not valid system.profile structure")
		return connection, false
	}

	return connection, true
}

func sendQuery(l row, logger *logrus.Entry, docsChan chan proto.SystemProfile, connections map[string]string) {
	var stats systemProfile
	err := json.Unmarshal(l.Attr, &stats)
	if err != nil {
		logger.Debugln("not valid system.profile structure")
		return
	}

	if strings.Contains(stats.Ns, ".$cmd") {
		logger.Debugln("skipping line with Ns .$cmd")
		return
	}

	fixOpAndNs(&stats)

	doc := stats.SystemProfile
	if user, ok := connections[stats.Remote]; ok {
		doc.User = user
	}
	doc.Client = strings.Split(stats.Remote, ":")[0]
	doc.Ts = l.T.Date

	var command bson.D
	for key, value := range stats.Command {
		command = append(command, bson.E{Key: key, Value: value})
	}

	doc.Command = command
	docsChan <- doc
}

// Stop stops monitor.
func (m *Monitor) Stop() {
	m.m.Lock()
	defer m.m.Unlock()

	if !m.running {
		return
	}

	err := m.reader.Close()
	if err != nil {
		m.logger.Error(err)
	}

	m.running = false
}
