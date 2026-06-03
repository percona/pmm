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

package channel

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	logshipv1 "github.com/percona/pmm/api/logship/v1"
)

const (
	logShipPrometheusSubsystem = "logship_channel"
)

// LogShipChannel encapsulates the client-streaming gRPC stream that ships client/database logs from
// pmm-agent to pmm-managed. It mirrors RTAChannel and shares the same underlying gRPC connection.
//
// All exported methods are thread-safe.
type LogShipChannel struct {
	s logshipv1.LogShipService_ShipClient
	l *logrus.Entry

	mSend prometheus.Counter

	sendM sync.Mutex

	closeOnce sync.Once
	closeWait chan struct{}
	closeErr  error
}

// NewLogShipChannel creates a new uni-directional log-shipping channel with the given stream.
//
// Stream should not be used by the caller after channel is created.
func NewLogShipChannel(stream logshipv1.LogShipService_ShipClient) *LogShipChannel {
	c := &LogShipChannel{
		s:         stream,
		l:         logrus.WithField("component", "logship_channel"),
		closeWait: make(chan struct{}),
		mSend: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: rtaPrometheusNamespace,
			Subsystem: logShipPrometheusSubsystem,
			Name:      "messages_sent_total",
			Help:      "A total number of shipped log messages to pmm-managed.",
		}),
	}

	go c.runHealthPing()

	return c
}

// Wait blocks until the channel is closed and returns the reason why it was closed.
func (c *LogShipChannel) Wait() error {
	<-c.closeWait
	return c.closeErr
}

// Send ships a message to pmm-managed. It is a no-op once the channel is closed (see Wait).
func (c *LogShipChannel) Send(msg *logshipv1.ShipRequest) {
	c.sendM.Lock()

	select {
	case <-c.closeWait:
		c.sendM.Unlock()
		return
	default:
	}

	err := c.s.Send(msg)
	c.sendM.Unlock()

	if err != nil {
		c.l.Errorf("Failed to send message: %+v", status.Code(err))
		c.close(fmt.Errorf("failed to send message: %w", err))
		return
	}

	c.mSend.Inc()
}

// Describe implements prometheus.Collector.
func (c *LogShipChannel) Describe(ch chan<- *prometheus.Desc) {
	c.mSend.Describe(ch)
}

// Collect implements prometheus.Collector.
func (c *LogShipChannel) Collect(ch chan<- prometheus.Metric) {
	c.mSend.Collect(ch)
}

// runHealthPing sends an empty ShipRequest periodically to keep the stream alive during quiet periods,
// the same way RTAChannel does.
func (c *LogShipChannel) runHealthPing() {
	pingReq := &logshipv1.ShipRequest{}

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeWait:
			return
		case <-ticker.C:
			c.Send(pingReq)
		}
	}
}

func (c *LogShipChannel) close(err error) {
	c.closeOnce.Do(func() {
		c.l.Debugf("Closing with error: %+v", err)
		c.closeErr = err

		c.sendM.Lock()
		_, closeErr := c.s.CloseAndRecv()
		if closeErr != nil {
			c.l.Errorf("Failed to receive final response: %v", closeErr)
		}

		close(c.closeWait)
		c.sendM.Unlock()
	})
}

// check interfaces.
var (
	_ prometheus.Collector = (*LogShipChannel)(nil)
)
