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

package collector

import (
	"context"
	"encoding/json"
	"time"

	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/percona/pmm/agent/utils/filereader"
)

// MongologReader read a MongoDB log file.
type MongologReader struct {
	logger   *logrus.Entry
	r        filereader.Reader
	docsChan chan<- proto.SystemProfile
	doneChan <-chan struct{}
}

const slowQuery = "Slow query"

// Mongolog is a helper structure to unmarshall Monglog row to system.Profile metrics.
type Mongolog struct {
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

// GetLogFilePath returns path for mongo log file.
func GetLogFilePath(client *mongo.Client) (string, error) {
	var result bson.M
	err := client.Database("admin").RunCommand(context.TODO(), bson.M{"getCmdLineOpts": 1}).Decode(&result)
	if err != nil {
		return "", errors.Wrap(err, "failed to run command getCmdLineOpts")
	}

	if parsed, ok := result["parsed"].(bson.M); ok {
		if systemLog, ok := parsed["systemLog"].(bson.M); ok {
			if logPath, ok := systemLog["path"].(string); ok {
				return logPath, nil
			}
		}
	}

	if argv, ok := result["argv"].([]interface{}); ok {
		for i := 0; i < len(argv); i++ {
			if arg, ok := argv[i].(string); ok && arg == "--logpath" && i+1 < len(argv) {
				if value, ok := argv[i+1].(string); ok {
					return value, nil
				}
			}
		}
	}

	return "", errors.New("no log path found, logs may be in Docker stdout")
}

// NewReader returns a new MongologReader that reads from the given reader.
func NewReader(docsChan chan<- proto.SystemProfile, doneChan <-chan struct{}, logsPath string, logger *logrus.Entry) (*MongologReader, error) {
	reader, err := filereader.NewContinuousFileReader(logsPath, logger)
	if err != nil {
		return nil, err
	}

	p := &MongologReader{
		logger:   logger,
		r:        reader,
		docsChan: docsChan,
		doneChan: doneChan,
	}

	return p, nil
}

// ReadFile continuously read new lines from file, until it is canceled or considered as done.
func (p *MongologReader) ReadFile(ctx context.Context) {
	p.logger.Debugln("reader started")
	for {
		select {
		case <-ctx.Done():
			p.logger.Debugln("context done")
			return
		case <-p.doneChan:
			p.logger.Debugln("reader done")
			return
		default:
			line, err := p.r.NextLine()
			if err != nil {
				p.logger.Error(err)
				return
			}
			p.logger.Debugf("readed line: %s", line)

			var l Mongolog
			var doc proto.SystemProfile
			if line == "" || !json.Valid([]byte(line)) {
				continue
			}
			err = json.Unmarshal([]byte(line), &l)
			if err != nil {
				p.logger.Error(err)
				continue
			}
			if l.Msg != slowQuery {
				continue
			}

			var stats systemProfile
			err = json.Unmarshal(l.Attr, &stats)
			if err != nil {
				p.logger.Debugln("not valid system.profile structure")
				continue
			}

			doc = stats.SystemProfile
			doc.Ts = l.T.Date

			var command bson.D
			for key, value := range stats.Command {
				command = append(command, bson.E{Key: key, Value: value})
			}

			doc.Command = command
			p.docsChan <- doc
		}
	}
}
