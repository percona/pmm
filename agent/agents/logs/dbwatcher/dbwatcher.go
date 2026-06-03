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

// Package dbwatcher runs a built-in agent that tails database log files and ships their lines to the
// PMM Server over the existing agent channel. Parsing of the raw lines happens centrally on the server.
package dbwatcher

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/utils/backoff"
	"github.com/percona/pmm/agent/utils/filereader"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	logshipv1 "github.com/percona/pmm/api/logship/v1"
)

const (
	backoffMinDelay = 1 * time.Second
	backoffMaxDelay = 30 * time.Second
)

// WatchedFile is a single log file to tail and its type.
type WatchedFile struct {
	Path string
	Type string // error, slow or general
}

// Params are the database log-watcher agent parameters.
type Params struct {
	AgentID     string
	ServiceID   string
	ServiceName string
	DBSystem    string // OTel db.system: mysql, postgresql, ...
	Files       []WatchedFile
	// AllowedDirs restricts which directories may be watched. Empty means only the explicitly
	// configured paths are allowed (no additional directories).
	AllowedDirs []string
}

// DBLogWatcher tails configured database log files and ships their lines.
type DBLogWatcher struct {
	params  *Params
	l       *logrus.Entry
	changes chan agents.Change
}

// New creates a new database log-watcher agent.
func New(params *Params, l *logrus.Entry) (*DBLogWatcher, error) {
	return &DBLogWatcher{
		params:  params,
		l:       l,
		changes: make(chan agents.Change, 100), //nolint:mnd
	}, nil
}

// Run tails the configured files until ctx is canceled.
func (s *DBLogWatcher) Run(ctx context.Context) {
	defer func() {
		s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE}
		close(s.changes)
	}()

	s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING}

	var wg sync.WaitGroup
	var watched int
	for _, f := range s.params.Files {
		path, err := s.validatePath(f.Path)
		if err != nil {
			s.l.Errorf("Refusing to watch %q: %s.", f.Path, err)
			continue
		}
		watched++
		wg.Add(1)
		go func(path, logType string) {
			defer wg.Done()
			s.watchFile(ctx, path, logType)
		}(path, f.Type)
	}

	if watched == 0 {
		s.l.Warn("No valid log files to watch.")
		s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_WAITING}
		<-ctx.Done()
		return
	}

	s.changes <- agents.Change{Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING}
	wg.Wait()
}

// watchFile tails a single file, reopening it (with backoff) until ctx is canceled.
func (s *DBLogWatcher) watchFile(ctx context.Context, path, logType string) {
	b := backoff.New(backoffMinDelay, backoffMaxDelay)
	for ctx.Err() == nil {
		reader, err := filereader.NewContinuousFileReader(path, s.l)
		if err != nil {
			s.l.Warnf("Cannot open %q: %s.", path, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(b.Delay()):
				continue
			}
		}
		b.Reset()
		s.tail(ctx, reader, logType)
	}
}

// tail reads lines until the reader is closed (on ctx cancellation) and ships each line.
func (s *DBLogWatcher) tail(ctx context.Context, reader *filereader.ContinuousFileReader, logType string) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = reader.Close()
		case <-done:
		}
	}()
	defer close(done)

	for {
		line, err := reader.NextLine()
		if err != nil {
			return
		}
		if line = strings.TrimRight(line, "\r\n"); line == "" {
			continue
		}
		s.ship(line, logType)
	}
}

func (s *DBLogWatcher) ship(line, logType string) {
	req := &logshipv1.ShipRequest{
		ServiceName: s.params.ServiceName,
		ResourceAttributes: map[string]string{
			"db.system":    s.params.DBSystem,
			"service.id":   s.params.ServiceID,
			"pmm.source":   "client",
			"pmm.agent_id": s.params.AgentID,
		},
		Records: []*logshipv1.LogRecord{{
			Time:       timestamppb.Now(),
			Body:       line,
			Attributes: map[string]string{"pmm.log_type": logType},
		}},
	}
	select {
	case s.changes <- agents.Change{LogShipRequests: []*logshipv1.ShipRequest{req}}:
	default:
		// drop under backpressure to never block tailing
	}
}

// validatePath resolves the path and enforces the allowlist, defeating symlink escapes.
func (s *DBLogWatcher) validatePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// The file may not exist yet; fall back to the cleaned absolute path for the allowlist check.
		resolved = filepath.Clean(abs)
	}

	if len(s.params.AllowedDirs) > 0 {
		allowed := false
		for _, dir := range s.params.AllowedDirs {
			if pathWithin(resolved, dir) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", errors.Errorf("path %q is not within an allowed directory", resolved)
		}
	}
	return resolved, nil
}

func pathWithin(path, dir string) bool {
	dir = filepath.Clean(dir)
	rel, err := filepath.Rel(dir, filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Changes returns the channel of agent changes.
func (s *DBLogWatcher) Changes() <-chan agents.Change {
	return s.changes
}

// Describe implements prometheus.Collector.
func (s *DBLogWatcher) Describe(chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector.
func (s *DBLogWatcher) Collect(chan<- prometheus.Metric) {}

// check interface.
var _ agents.BuiltinAgent = (*DBLogWatcher)(nil)
