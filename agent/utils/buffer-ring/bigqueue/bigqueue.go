// Copyright 2023 Percona LLC
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

// Package bigqueue implements ring buffer based on bigqueue.
package bigqueue

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jhunters/bigqueue"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/percona/pmm/agent/models"
	agenterrors "github.com/percona/pmm/agent/utils/errors"
	"github.com/percona/pmm/api/agentpb"
)

const (
	metaFileSize   = 16 + 8 // represent FrontFileInfo size + MetaFileInfo size
	indexEntrySize = 32     // represent index entry size
)

var (
	ErrClosed = errors.New("cache closed")

	dataPageSize   = 1024 * 1024                   // represent page size for data entries
	indexPageSize  = dataPageSize / indexEntrySize // represent page size for index entries (default bigqueue ratio)
	drainThreshold = int64(1024 * 1024)            // represent threshold for preliminary draining
	gcDuration     = 10 * time.Second              // represent gc ticker duration
)

// Ring represent ring buffer based on bigqueue.
type Ring struct {
	l  *logrus.Entry
	fq *bigqueue.FileQueue
	wg sync.WaitGroup

	sendLock  sync.Mutex
	recvLock  sync.Mutex
	totalSize int64 // represent the limit after which old data will be overwritten

	sender atomic.Pointer[models.Sender]

	gcCh         chan struct{}
	drainCh      chan struct{}
	recvNotifyCh chan struct{}
	establishCh  chan struct{}
	done         chan struct{}
}

// New creates/loads ring buffer.
func New(dir string, size uint32, l *logrus.Entry) (*Ring, error) {
	err := initPaths(dir)
	if err != nil {
		return nil, err
	}
	dir, queueName := filepath.Split(dir)
	if lastRuneIdx := len(dir) - 1; len(dir) > 0 && rune(dir[lastRuneIdx]) == filepath.Separator {
		dir = dir[:lastRuneIdx]
	}
	if metaSize := uint32(metaFileSize + indexPageSize + dataPageSize); metaSize > size {
		return nil, fmt.Errorf("cache size must be greater than '%d' bytes to store at least one entry", metaSize)
	}
	fq := &bigqueue.FileQueue{}
	if err = fq.Open(dir, queueName, &bigqueue.Options{
		DataPageSize:      dataPageSize,
		IndexItemsPerPage: int(math.Log2(float64(indexPageSize) / indexEntrySize)),
	}); err != nil {
		return nil, err
	}
	out := &Ring{
		l:            l,
		fq:           fq,
		totalSize:    int64(size),
		drainCh:      make(chan struct{}, 1),
		gcCh:         make(chan struct{}, 1),
		establishCh:  make(chan struct{}, 1),
		recvNotifyCh: make(chan struct{}, 1),
		done:         make(chan struct{}),
	}
	out.gcRunner()
	out.sendRunner()
	if !out.isEmpty() {
		asyncNotify(out.recvNotifyCh)
	}
	return out, nil
}

// Send stores agent responses in cache on nil channel.
func (r *Ring) Send(resp *models.AgentResponse) error {
	msg := &agentpb.AgentMessage{Id: resp.ID}
	if resp.Payload != nil {
		msg.Payload = resp.Payload.AgentMessageResponsePayload()
	}
	if resp.Status != nil {
		msg.Status = resp.Status.Proto()
	}

	var (
		err error
		s   = r.sender.Load()
	)

	r.recvLock.Lock()
	defer r.recvLock.Unlock()
	if r.isEmpty() && s != nil {
		err = (*s).Send(resp)
		if err != nil && errors.As(err, &agenterrors.ErrChanConn) {
			if r.sender.CompareAndSwap(s, nil) {
				asyncRelease(r.establishCh)
				r.l.Debugf("sender released: %v", err)
			}
		} else {
			return err
		}
	}

	r.push(msg)
	return nil
}

// SendAndWaitResponse stores AgentMessageRequestPayload on nil channel.
func (r *Ring) SendAndWaitResponse(payload agentpb.AgentRequestPayload) (agentpb.ServerResponsePayload, error) { //nolint:unparam,ireturn
	var (
		err  error
		resp agentpb.ServerResponsePayload
		s    = r.sender.Load()
	)

	r.recvLock.Lock()
	defer r.recvLock.Unlock()
	if r.isEmpty() && s != nil {
		resp, err = (*s).SendAndWaitResponse(payload)
		if err != nil && errors.As(err, &agenterrors.ErrChanConn) {
			if r.sender.CompareAndSwap(s, nil) {
				asyncRelease(r.establishCh)
				r.l.Debugf("sender released: %v", err)
			}
		} else {
			return resp, err
		}
	}

	r.push(&agentpb.AgentMessage{Payload: payload.AgentMessageRequestPayload()})
	return &agentpb.StateChangedResponse{}, nil
}

// SetSender check and set sender and notify sender loop.
func (r *Ring) SetSender(s models.Sender) {
	r.sender.Store(&s)
	asyncNotify(r.establishCh)
	r.l.Debug("sender set")
}

// Close closes cache.
func (r *Ring) Close() {
	select {
	case <-r.done:
	default:
		close(r.done)
		r.wg.Wait()
		r.recvLock.Lock()
		err := r.fq.Close()
		r.recvLock.Unlock()
		if err != nil {
			r.l.Errorf("closing cache: %+v", err)
		}
		r.l.Info("cache closed")
	}
}

func (r *Ring) isEmpty() bool {
	r.recvLock.Lock()
	r.recvLock.Unlock()
	return r.fq.IsEmpty()
}

func (r *Ring) push(msg *agentpb.AgentMessage) {
	b, err := proto.Marshal(msg)
	if err != nil {
		r.l.Errorf("marshal proto while inserting message to cache: %+v", err)
		return
	}
	size := int64(len(b)) + indexEntrySize
	if size > r.totalSize {
		r.l.Errorf("data size: '%d' overflows free cache space: '%d'", size, r.totalSize)
		return
	}
	select {
	case <-r.done:
		return
	default:
	}
	r.recvLock.Lock()
	_, err = r.fq.Enqueue(b)
	r.recvLock.Unlock()
	if err != nil {
		r.l.Errorf("inserting to cache: %+v", err)
	}
	asyncNotify(r.recvNotifyCh)
}

func (r *Ring) gcRunner() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(gcDuration)
		defer ticker.Stop()
		for {
			select {
			case <-r.done:
				r.doDrain()
				return
			case <-r.drainCh:
				r.doDrain()
			case <-ticker.C:
				r.doDrain()
			case <-r.gcCh:
				r.sendLock.Lock()
				r.runGC()
				r.sendLock.Unlock()
			}
		}
	}()
}

func (r *Ring) doDrain() {
	if overflow := r.size() + drainThreshold - r.totalSize; overflow > 0 {
		r.sendLock.Lock()
		r.drain(overflow)
		r.runGC()
		r.sendLock.Unlock()
	}
}

func (r *Ring) sendRunner() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.done:
				return
			case <-r.recvNotifyCh:
				r.sendInLoop()
			}
		}
	}()
}

func (r *Ring) sendInLoop() {
	var s *models.Sender
	for {
		s = r.sender.Load()
		if s != nil {
			break
		}
		select {
		case <-r.done:
			return
		case <-r.establishCh:
			continue
		}
	}
	r.sendLock.Lock()
	defer r.sendLock.Unlock()
	var count int
	for {
		select {
		case <-r.done:
			return
		default:
		}
		r.recvLock.Lock()
		_, b, err := r.fq.Peek()
		r.recvLock.Unlock()
		if err != nil {
			r.l.Errorf("reading entry from cache: %+v", err)
		}
		if len(b) == 0 {
			break
		}
		var m agentpb.AgentMessage
		if err := proto.Unmarshal(b, &m); err != nil {
			r.l.Errorf("unmarshal entry from cache: %+v", err)
		} else if err = r.send(*s, &m); err != nil {
			if r.sender.CompareAndSwap(s, nil) {
				asyncRelease(r.establishCh)
				r.l.Debugf("sender released: %v", err)
			}
			break
		}
		r.recvLock.Lock()
		r.fq.Skip(1) //nolint:errcheck
		r.recvLock.Unlock()
		count++
	}
	if count > 0 {
		asyncNotify(r.gcCh)
	}
}

// initPaths creates all paths for queue to use. Original repo creates directories with perm error.
func initPaths(dir string) error {
	for _, path := range []string{
		"",
		bigqueue.IndexFileName,
		bigqueue.DataFileName,
		bigqueue.MetaFileName,
		bigqueue.FrontFileName,
	} {
		if err := os.MkdirAll(filepath.Join(dir, path), os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (r *Ring) drain(amount int64) {
	for size := int64(0); size < amount; {
		r.recvLock.Lock()
		_, b, err := r.fq.Dequeue()
		r.recvLock.Unlock()
		if err != nil {
			r.l.Errorf("draining cache: %+v", err)
			return
		}
		if len(b) == 0 {
			return
		}
		size += int64(len(b)) + indexEntrySize
	}
}

func (r *Ring) size() int64 {
	r.recvLock.Lock()
	status := r.fq.Status()
	r.recvLock.Unlock()
	sum := status.FrontFileInfo.Size + status.MetaFileInfo.Size
	for _, list := range status.IndexFileList {
		sum += list.Size
	}
	for _, list := range status.DataFileList {
		sum += list.Size
	}
	return sum
}

func (r *Ring) runGC() {
	r.recvLock.Lock()
	defer r.recvLock.Unlock()
	if err := r.fq.Gc(); err != nil {
		r.l.Errorf("run gc: %+v", err)
	}
}

func (r *Ring) send(s models.Sender, m *agentpb.AgentMessage) error {
	var err error
	switch p := m.Payload.(type) {
	// responses
	case *agentpb.AgentMessage_StartAction:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.StartAction})
	case *agentpb.AgentMessage_StopAction:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.StopAction})
	case *agentpb.AgentMessage_PbmSwitchPitr:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.PbmSwitchPitr})
	case *agentpb.AgentMessage_StartJob:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.StartJob})
	case *agentpb.AgentMessage_JobStatus:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.JobStatus})
	case *agentpb.AgentMessage_GetVersions:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.GetVersions})
	case *agentpb.AgentMessage_JobProgress:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.JobProgress})
	case *agentpb.AgentMessage_StopJob:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.StopJob})
	case *agentpb.AgentMessage_CheckConnection:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.CheckConnection})
	case *agentpb.AgentMessage_JobResult:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.JobResult})
	case *agentpb.AgentMessage_AgentLogs:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.AgentLogs})
	case *agentpb.AgentMessage_SetState:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.SetState})
	case *agentpb.AgentMessage_Pong:
		err = s.Send(&models.AgentResponse{ID: m.Id, Status: grpcstatus.FromProto(m.Status), Payload: p.Pong})
	// requests
	case *agentpb.AgentMessage_ActionResult:
		_, err = s.SendAndWaitResponse(p.ActionResult)
	case *agentpb.AgentMessage_QanCollect:
		_, err = s.SendAndWaitResponse(p.QanCollect)
	case *agentpb.AgentMessage_StateChanged:
		_, err = s.SendAndWaitResponse(p.StateChanged)
	default:
		r.l.Errorf("unknown message: %T", m)
		return nil
	}
	if err != nil && errors.As(err, &agenterrors.ErrChanConn) {
		return err
	}
	return nil
}

func asyncNotify(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func asyncRelease(ch chan struct{}) {
	select {
	case <-ch:
	default:
	}
}
