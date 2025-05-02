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
	"sync"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/percona/pmm/agent/utils/filereader"
)

const (
	collectorChanCapacity = 100
	collectorWaitDuration = time.Second
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
func (m *Monitor) Start(ctx context.Context, docsChan chan proto.SystemProfile, doneChan <-chan struct{}, wg *sync.WaitGroup) error {
	m.m.Lock()
	defer m.m.Unlock()

	if m.running {
		return nil
	}

	go func() {
		readFile(ctx, m.reader, docsChan, doneChan, wg, m.logger)
		m.logger.Debugln("reading routine quit")
	}()
	m.running = true

	return nil
}

const slowQuery = "Slow query"

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
}

// readFile continuously read new lines from file, until it is canceled or considered as done.
func readFile(ctx context.Context, reader *filereader.ContinuousFileReader, docsChan chan proto.SystemProfile, doneChan <-chan struct{}, wg *sync.WaitGroup, logger *logrus.Entry) {
	defer wg.Done()
	logger.Debugln("reader started")
	for {
		select {
		case <-ctx.Done():
			logger.Debugln("context done")
			return
		case <-doneChan:
			logger.Debugln("reader done")
			return
		default:
			logger.Debugln("reading line")
			fmt.Println("readning line")
			line, err := reader.NextLine()
			if err != nil {
				logger.Error(err)
				return
			}
			fmt.Println("readed")
			logger.Debugf("readed line: %s", line)

			var l row
			var doc proto.SystemProfile
			if line == "" || !json.Valid([]byte(line)) {
				continue
			}
			err = json.Unmarshal([]byte(line), &l)
			if err != nil {
				logger.Error(err)
				continue
			}
			if l.Msg != slowQuery {
				continue
			}

			var stats systemProfile
			err = json.Unmarshal(l.Attr, &stats)
			if err != nil {
				logger.Debugln("not valid system.profile structure")
				continue
			}

			doc = stats.SystemProfile
			doc.Ts = l.T.Date

			var command bson.D
			for key, value := range stats.Command {
				command = append(command, bson.E{Key: key, Value: value})
			}

			doc.Command = command
			docsChan <- doc
		}
	}
}

// Stop stops monitor.
func (m *Monitor) Stop() {
	m.m.Lock()
	defer m.m.Unlock()

	if !m.running {
		return
	}

	m.running = false
}
