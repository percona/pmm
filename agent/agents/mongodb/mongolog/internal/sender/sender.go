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

package sender

import (
	"context"
	"runtime/pprof"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/agents/mongodb/mongolog/internal/report"
)

// New creates new sender.
func New(reportChan <-chan *report.Report, w Writer, logger *logrus.Entry) *Sender {
	return &Sender{
		reportChan: reportChan,
		w:          w,
		logger:     logger,
	}
}

// Sender represents report channel and writer.
type Sender struct {
	// dependencies
	reportChan <-chan *report.Report
	w          Writer
	logger     *logrus.Entry

	// state
	m        sync.Mutex      // Lock() to protect internal consistency of the service
	running  bool            // Is this service running?
	doneChan chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg       *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Start starts but doesn't wait until it exits.
func (s *Sender) Start() error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.running {
		return nil
	}

	// create new channels over which we will communicate to...
	// ... inside goroutine to close it
	s.doneChan = make(chan struct{})

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	s.wg = &sync.WaitGroup{}
	s.wg.Add(1)

	ctx := context.Background()
	labels := pprof.Labels("component", "mongodb.sender")
	go pprof.Do(ctx, labels, func(ctx context.Context) {
		start(ctx, s.wg, s.reportChan, s.w, s.logger, s.doneChan)
	})

	s.running = true
	return nil
}

// Stop stops running sender.
func (s *Sender) Stop() {
	s.m.Lock()
	defer s.m.Unlock()
	if !s.running {
		return
	}
	s.running = false

	// notify goroutine to close
	close(s.doneChan)

	// wait for goroutines to exit
	s.wg.Wait()
}

func start(ctx context.Context, wg *sync.WaitGroup, reportChan <-chan *report.Report, w Writer, logger *logrus.Entry, doneChan <-chan struct{}) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-doneChan:
			return
		case report, ok := <-reportChan:
			// if channel got closed we should exit as there is nothing we can listen to
			if !ok {
				return
			}

			// sent report
			if err := w.Write(report); err != nil {
				logger.Warn("Lost report:", err)
				continue
			}
		}
	}
}

// Writer write QAN Report
type Writer interface {
	Write(r *report.Report) error
}
