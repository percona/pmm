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
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	// Interval between health ping messages.
	// Should be less than 1 minute to prevent gRPC stream from being closed
	// by underlying layers due to inactivity.
	pingInterval = 30 * time.Second

	rtaPrometheusNamespace = "pmm_agent"
	rtaPrometheusSubsystem = "rta_channel"
)

// RTAChannel encapsulates client-streaming gRPC stream from pmm-agent to pmm-managed.
//
// All exported methods are thread-safe.
type RTAChannel struct { //nolint:maligned
	s rtav1.CollectorService_CollectClient
	l *logrus.Entry

	// sent messages counter (for prometheus)
	mSend prometheus.Counter

	// protects access to `s` stream
	sendM sync.Mutex

	closeOnce sync.Once
	closeWait chan struct{}
	closeErr  error
}

// NewRTAChannel creates new uni-directional communication channel with given stream.
//
// Stream should not be used by the caller after channel is created.
func NewRTAChannel(stream rtav1.CollectorService_CollectClient) *RTAChannel {
	s := &RTAChannel{
		s:         stream,
		l:         logrus.WithField("component", "rta_channel"), // only for debug logging
		closeWait: make(chan struct{}),
		mSend: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: rtaPrometheusNamespace,
			Subsystem: rtaPrometheusSubsystem,
			Name:      "rta_messages_sent_total",
			Help:      "A total number of sent Real-Time Analytics messages to pmm-managed.",
		}),
	}

	go s.runHealthPing()

	return s
}

// runHealthPing sends empty rtav1.CollectRequest{} as health ping periodically.
func (c *RTAChannel) runHealthPing() {
	// It is required to send something periodically to keep the stream alive.
	// As soon as stream for sending RTA data is created always, there are
	// situations when no RTA data is sent for a long time because no RTA agents are
	// requested to be running.
	// In such a situation, to keep the stream alive, we send empty
	// rtav1.CollectRequest{} messages as health pings.
	// Otherwise, the stream is closed by underlying gRPC layers after 1 minute of inactivity.
	// Keep-alive pings on lower layers (TCP, HTTP/2, gRPC) are not sufficient because they
	// are applied to the whole connection.
	pingReq := &rtav1.CollectRequest{Queries: []*rtav1.QueryData{}}

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

// close marks channel as closed with given error - only once.
func (c *RTAChannel) close(err error) {
	c.closeOnce.Do(func() {
		c.l.Debugf("Closing with error: %+v", err)
		c.closeErr = err

		c.sendM.Lock()
		// Close stream and receive final response
		_, closeErr := c.s.CloseAndRecv()
		if closeErr != nil {
			c.l.Errorf("Failed to receive final response: %v", closeErr)
		}

		close(c.closeWait)
		c.sendM.Unlock()
	})
}

// Wait blocks until channel is closed and returns the reason why it was closed.
//
// When Wait returns, underlying gRPC connection should be terminated to prevent goroutine leak.
func (c *RTAChannel) Wait() error {
	<-c.closeWait
	return c.closeErr
}

// Send sends message to pmm-managed. It is no-op once channel is closed (see Wait).
func (c *RTAChannel) Send(msg *rtav1.CollectRequest) {
	c.sendM.Lock()

	select {
	case <-c.closeWait:
		return
	default:
	}

	// Check log level before calling formatting function.
	// Do not waste resources in case debug level is not enabled.
	if c.l.Logger.IsLevelEnabled(logrus.DebugLevel) {
		// do not use default compact representation for large/complex messages
		if size := proto.Size(msg); size < 100 { //nolint:mnd
			c.l.Debugf("Sending message (%d bytes): %s.", size, msg)
		} else {
			c.l.Debugf("Sending message (%d bytes):\n%s\n", size, prototext.Format(msg))
		}
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
func (c *RTAChannel) Describe(ch chan<- *prometheus.Desc) {
	c.mSend.Describe(ch)
}

// Collect implement prometheus.Collector.
func (c *RTAChannel) Collect(ch chan<- prometheus.Metric) {
	c.mSend.Collect(ch)
}

// check interfaces.
var (
	_ prometheus.Collector = (*RTAChannel)(nil)
)
