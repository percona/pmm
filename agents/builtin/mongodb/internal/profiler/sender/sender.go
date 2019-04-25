// pmm-agent
// Copyright (C) 2018 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package sender

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-agent/agents/builtin/mongodb/internal/report"
	"github.com/percona/pmm-agent/agents/builtin/mongodb/internal/status"
)

func New(reportChan <-chan *report.Report, w Writer, logger *logrus.Entry) *Sender {
	return &Sender{
		reportChan: reportChan,
		w:          w,
		logger:     logger,
	}
}

type Sender struct {
	// dependencies
	reportChan <-chan *report.Report
	w          Writer
	logger     *logrus.Entry

	// stats
	status *status.Status

	// state
	sync.RWMutex                 // Lock() to protect internal consistency of the service
	running      bool            // Is this service running?
	doneChan     chan struct{}   // close(doneChan) to notify goroutines that they should shutdown
	wg           *sync.WaitGroup // Wait() for goroutines to stop after being notified they should shutdown
}

// Start starts but doesn't wait until it exits
func (self *Sender) Start() error {
	self.Lock()
	defer self.Unlock()
	if self.running {
		return nil
	}

	// create new channels over which we will communicate to...
	// ... inside goroutine to close it
	self.doneChan = make(chan struct{})

	// set stats
	stats := &stats{}
	self.status = status.New(stats)

	// start a goroutine and Add() it to WaitGroup
	// so we could later Wait() for it to finish
	self.wg = &sync.WaitGroup{}
	self.wg.Add(1)
	go start(self.wg, self.reportChan, self.w, self.logger, self.doneChan, stats)

	self.running = true
	return nil
}

// Stop stops running
func (self *Sender) Stop() {
	self.Lock()
	defer self.Unlock()
	if !self.running {
		return
	}
	self.running = false

	// notify goroutine to close
	close(self.doneChan)

	// wait for goroutines to exit
	self.wg.Wait()
	return
}

func (self *Sender) Status() map[string]string {
	self.RLock()
	defer self.RUnlock()
	if !self.running {
		return nil
	}

	return self.status.Map()
}

func (self *Sender) Name() string {
	return "sender"
}

func start(wg *sync.WaitGroup, reportChan <-chan *report.Report, w Writer, logger *logrus.Entry, doneChan <-chan struct{}, stats *stats) {
	// signal WaitGroup when goroutine finished
	defer wg.Done()

	for {

		select {
		case report, ok := <-reportChan:
			stats.In += 1
			// if channel got closed we should exit as there is nothing we can listen to
			if !ok {
				return
			}

			// check if we should shutdown
			select {
			case <-doneChan:
				return
			default:
				// just continue if not
			}

			// sent report
			if err := w.Write(report); err != nil {
				stats.ErrIter += 1
				logger.Warn("Lost report:", err)
				continue
			}
			stats.Out += 1
		case <-doneChan:
			return
		}
	}
}

// Writer write QAN Report
type Writer interface {
	Write(r *report.Report) error
}
