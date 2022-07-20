// pmm-agent
// Copyright 2019 Percona LLC
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

package connectionuptime

import (
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type bitSet struct {
	bigInt big.Int
}

func (bs *bitSet) SetBit(i int, b uint) *bitSet {
	bs.bigInt = *bs.bigInt.SetBit(&bs.bigInt, i, b)
	return bs
}

func (bs *bitSet) Bit(i int) uint {
	return bs.bigInt.Bit(i)
}

// Service calculates connection uptime between agent and server
type Service struct {
	uptimeSeconds bitSet

	windowPeriodSeconds int64
	indexLastStatus     int64
	startTime           time.Time
	lastStatusTimestamp time.Time

	mx sync.Mutex

	l *logrus.Entry
}

// NewService creates new instance of Service
func NewService(windowPeriod time.Duration) *Service {
	return &Service{
		windowPeriodSeconds: int64(windowPeriod.Seconds()),
		l:                   logrus.WithField("component", "connection-uptime-service"),
		uptimeSeconds:       bitSet{},
	}
}

// RegisterConnectionStatus adds new connection status
func (s *Service) RegisterConnectionStatus(timestamp time.Time, connected bool) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if err := s.registerConnectionStatus(timestamp, connected); err != nil {
		// only print error here
		s.l.Error(err.Error())
	}
}

func (s *Service) registerConnectionStatus(timestamp time.Time, connected bool) error {
	if s.startTime.IsZero() {
		s.startTime = timestamp
		s.lastStatusTimestamp = timestamp
		s.uptimeSeconds.SetBit(0, toUint(connected))
		s.indexLastStatus = 0

		return nil
	}

	endIndex, err := s.fillBitSetWithStatusUntilTimestamp(timestamp)
	if err != nil {
		return err
	}

	err = s.setLastStatusBitByIndex(endIndex, connected)
	if err != nil {
		return err
	}

	s.lastStatusTimestamp = timestamp
	return nil
}

func (s *Service) setLastStatusBitByIndex(endIndex int64, connected bool) error {
	s.indexLastStatus = endIndex % s.windowPeriodSeconds
	if s.indexLastStatus > math.MaxInt32 {
		return errors.Errorf("Index is higher then max int32 value: %d", s.indexLastStatus)
	}

	s.uptimeSeconds.SetBit(int(s.indexLastStatus), toUint(connected))
	return nil
}

func (s *Service) fillBitSetWithStatusUntilTimestamp(timestamp time.Time) (int64, error) {
	secondsFromLastEvent := timestamp.Unix() - s.lastStatusTimestamp.Unix()
	endIndex := s.indexLastStatus + secondsFromLastEvent
	lastConnectedStatusBit := s.uptimeSeconds.Bit(int(s.indexLastStatus))

	for i := s.indexLastStatus + 1; i < endIndex; i++ {
		// set the same status to elements of previous connection status
		index := i % s.windowPeriodSeconds
		if index > math.MaxInt32 {
			return 0, errors.Errorf("Index is higher then max int32 value: %d", index)
		}
		s.uptimeSeconds.SetBit(int(index), lastConnectedStatusBit)
	}
	return endIndex, nil
}

func toUint(b bool) uint {
	if b {
		return 1
	}
	return 0
}

// GetConnectedUpTimeSince calculates connected uptime between agent and server
// based on the connection statuses
func (s *Service) GetConnectedUpTimeSince(toTime time.Time) float32 {
	err := s.fillStatusesUntil(toTime)
	if err != nil {
		s.l.Error(err.Error())
		return 0
	}

	res, err := s.calculateConnectionUpTime(toTime)
	if err != nil {
		s.l.Error(err.Error())
		return 0
	}
	return res
}

func (s *Service) calculateConnectionUpTime(toTime time.Time) (float32, error) {
	numOfSeconds := s.getNumOfSecondsForCalculationUptime(toTime)
	startIndex := s.getStartIndex(numOfSeconds)
	connectedSeconds, err := s.getNumOfConnectedSeconds(startIndex, numOfSeconds)
	if err != nil {
		return 0, err
	}
	return float32(connectedSeconds) / float32(numOfSeconds) * 100, nil
}

func (s *Service) getNumOfSecondsForCalculationUptime(toTime time.Time) int64 {
	numOfSeconds := s.windowPeriodSeconds
	diffInSecondsBetweenStartTimeAndToTime := toTime.Unix() - s.startTime.Unix()
	if diffInSecondsBetweenStartTimeAndToTime < s.windowPeriodSeconds {
		numOfSeconds = diffInSecondsBetweenStartTimeAndToTime
	}
	return numOfSeconds
}

func (s *Service) getStartIndex(size int64) int64 {
	startElement := s.indexLastStatus - size
	if startElement < 0 {
		startElement = s.windowPeriodSeconds + startElement
	}
	return startElement
}

func (s *Service) getNumOfConnectedSeconds(startIndex int64, totalNumOfSeconds int64) (int, error) {
	endIndex := startIndex + totalNumOfSeconds
	connectedSeconds := 0
	for i := startIndex; i < endIndex; i++ {
		index := i % s.windowPeriodSeconds
		if index > math.MaxInt32 {
			return 0, errors.Errorf("Index is higher then max int32 value: %d", index)
		}

		if s.uptimeSeconds.Bit(int(index)) == 1 {
			connectedSeconds++
		}
	}
	return connectedSeconds, nil
}

// fill values in the slice until toTime
func (s *Service) fillStatusesUntil(toTime time.Time) error {
	if s.indexLastStatus > math.MaxInt32 {
		return errors.Errorf("indexLastStatus is higher then max int32 value: %d", s.indexLastStatus)
	}

	return s.registerConnectionStatus(toTime, s.uptimeSeconds.Bit(int(s.indexLastStatus)) == 1)
}
