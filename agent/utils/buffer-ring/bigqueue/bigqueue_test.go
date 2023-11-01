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

package bigqueue

import (
	"bytes"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/agent/models"
	"github.com/percona/pmm/api/agentpb"
)

func TestMetaSizes(t *testing.T) { //nolint:tparallel
	indexPageSize = 20
	t.Run("meta file size", func(t *testing.T) {
		t.Parallel()
		ring, err := New(filepath.Join(os.TempDir(), newRandomString(10)), uint32(dataPageSize+indexPageSize+metaFileSize), nil)
		assert.NoError(t, err)
		status := ring.fq.Status()
		assert.Equal(t, int64(metaFileSize), status.FrontFileInfo.Size+status.MetaFileInfo.Size)
	})
	t.Run("index entry size", func(t *testing.T) {
		t.Parallel()
		ring, err := New(filepath.Join(os.TempDir(), newRandomString(10)), uint32(dataPageSize+indexPageSize+metaFileSize), nil)
		assert.NoError(t, err)
		_, err = ring.fq.Enqueue([]byte("1"))
		assert.NoError(t, err)
		assert.Equal(t, int64(indexEntrySize), ring.fq.Status().IndexFileList[0].Size)
	})
}

func TestNew(t *testing.T) { //nolint:tparallel
	indexPageSize = 20
	t.Run("new with size less than meta", func(t *testing.T) {
		t.Parallel()
		_, err := New(filepath.Join(os.TempDir(), newRandomString(10)), uint32(dataPageSize+indexPageSize+metaFileSize)-1, nil)
		assert.Error(t, err)
	})
	t.Run("data size too big", func(t *testing.T) {
		t.Parallel()
		ring, log, cleanup := setupTest(t, filepath.Join(os.TempDir(), newRandomString(10)), uint32(dataPageSize+indexPageSize+metaFileSize))
		t.Cleanup(cleanup)
		_, err := ring.SendAndWaitResponse(&agentpb.QANCollectRequest{MetricsBucket: []*agentpb.MetricsBucket{{
			Common: &agentpb.MetricsBucket_Common{Queryid: newRandomString(dataPageSize + indexPageSize + metaFileSize)},
		}}})
		assert.NoError(t, err)
		assert.Equal(t, "level=error msg=\"data size: '1048668' overflows free cache space: '1048620'\" cache=test\n", log.String())
	})
}

type sender struct {
	t *testing.T
	i uint32
}

func (s *sender) Send(resp *models.AgentResponse) error { return nil }
func (s *sender) SendAndWaitResponse(payload agentpb.AgentRequestPayload) (agentpb.ServerResponsePayload, error) {
	qan, ok := payload.(*agentpb.QANCollectRequest)
	assert.Equal(s.t, true, ok)
	assert.Equal(s.t, 1, len(qan.MetricsBucket))
	assert.Equal(s.t, atomic.LoadUint32(&s.i), qan.MetricsBucket[0].Common.PlaceholdersCount)
	atomic.AddUint32(&s.i, 1)
	return nil, nil
}

func TestDrain(t *testing.T) { //nolint:tparallel
	dataPageSize = indexEntrySize
	indexPageSize = dataPageSize
	payloadLen := indexEntrySize - 11 // queryId + proto = 32
	drainThreshold = 0

	t.Run("push", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(os.TempDir(), newRandomString(10))
		ring, log, cleanup := setupTest(t, dir, uint32(dataPageSize+indexPageSize)*3+metaFileSize)
		t.Cleanup(cleanup)

		for i := uint32(1); i <= 4; i++ {
			_, err := ring.SendAndWaitResponse(&agentpb.QANCollectRequest{MetricsBucket: []*agentpb.MetricsBucket{{
				Common: &agentpb.MetricsBucket_Common{PlaceholdersCount: i, Queryid: newRandomString(payloadLen)},
			}}})
			assert.NoError(t, err)
			runtime.Gosched()
		}
		asyncNotify(ring.drainCh)
		runtime.Gosched()
		time.Sleep(1 * time.Second)
		s := sender{
			i: uint32(2), // first must be drained
			t: t,
		}
		ring.SetSender(&s)
		time.Sleep(1 * time.Second)
		assert.NotEqual(t, uint32(2), atomic.LoadUint32(&s.i))
		assert.Equal(t, -1, strings.LastIndex(log.String(), "level=error"))
	})
	t.Run("shutdown", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(os.TempDir(), newRandomString(10))
		ring, log, _ := setupTest(t, dir, uint32(dataPageSize+indexPageSize)*3+metaFileSize)
		for i := uint32(1); i <= 4; i++ {
			_, err := ring.SendAndWaitResponse(&agentpb.QANCollectRequest{MetricsBucket: []*agentpb.MetricsBucket{{
				Common: &agentpb.MetricsBucket_Common{PlaceholdersCount: i, Queryid: newRandomString(payloadLen)},
			}}})
			assert.NoError(t, err)
			runtime.Gosched()
		}
		time.Sleep(1 * time.Second)
		ring.Close()
		assert.Equal(t, -1, strings.LastIndex(log.String(), "closing cache"))

		ring, log, cleanup := setupTest(t, dir, uint32(dataPageSize+indexPageSize)*3+metaFileSize)
		t.Cleanup(cleanup)
		s := sender{
			i: uint32(2), // first must be drained
			t: t,
		}
		ring.SetSender(&s)
		time.Sleep(1 * time.Second)
		assert.NotEqual(t, uint32(2), atomic.LoadUint32(&s.i))
		assert.Equal(t, -1, strings.LastIndex(log.String(), "level=error"))
	})
	t.Run("size", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(os.TempDir(), newRandomString(10))
		ring, log, cleanup := setupTest(t, dir, uint32(dataPageSize+indexPageSize)*4+metaFileSize)
		t.Cleanup(cleanup)
		for i := uint32(1); i <= 5; i++ {
			_, err := ring.SendAndWaitResponse(&agentpb.QANCollectRequest{MetricsBucket: []*agentpb.MetricsBucket{{
				Common: &agentpb.MetricsBucket_Common{PlaceholdersCount: i, Queryid: newRandomString(payloadLen)},
			}}})
			assert.NoError(t, err)
			runtime.Gosched()
		}
		asyncNotify(ring.drainCh)
		runtime.Gosched()
		time.Sleep(1 * time.Second)

		// after push all messages
		size, err := dirSize(dir)
		assert.NoError(t, err)
		assert.Equal(t, int64(344), size)
		s := sender{
			i: uint32(2), // first must be drained
			t: t,
		}
		ring.SetSender(&s)
		time.Sleep(1 * time.Second)
		assert.NotEqual(t, uint32(2), atomic.LoadUint32(&s.i))
		assert.Equal(t, -1, strings.LastIndex(log.String(), "level=error"))

		// after send
		size, err = dirSize(dir)
		assert.NoError(t, err)
		assert.Equal(t, int64(indexPageSize+dataPageSize+metaFileSize), size)
	})
}

func TestReadWrite(t *testing.T) { //nolint:tparallel
	dataPageSize = indexEntrySize
	indexPageSize = dataPageSize
	payloadLen := indexEntrySize - 11 // queryId + proto = 32
	drainThreshold = 0

	t.Run("async read write", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(os.TempDir(), newRandomString(10))
		ring, log, cleanup := setupTest(t, dir, uint32(dataPageSize+indexPageSize)*10+metaFileSize)
		t.Cleanup(cleanup)

		started := make(chan struct{})
		go func() {
			close(started)
			for i := uint32(1); i <= 10; i++ {
				_, err := ring.SendAndWaitResponse(&agentpb.QANCollectRequest{MetricsBucket: []*agentpb.MetricsBucket{{
					Common: &agentpb.MetricsBucket_Common{PlaceholdersCount: i, Queryid: newRandomString(payloadLen)},
				}}})
				assert.NoError(t, err)
				runtime.Gosched()
			}
		}()
		<-started
		s := sender{
			i: uint32(1),
			t: t,
		}
		ring.SetSender(&s)
		time.Sleep(1 * time.Second)
		assert.NotEqual(t, uint32(1), atomic.LoadUint32(&s.i))
		assert.Equal(t, -1, strings.LastIndex(log.String(), "level=error"))
	})
}

func newRandomString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	const alp = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = alp[r.Intn(len(alp))]
	}
	return string(b)
}

func setupTest(t *testing.T, dir string, size uint32) (*Ring, *bytes.Buffer, func()) {
	t.Helper()
	var buf bytes.Buffer
	testLogger := logrus.Logger{
		Out:   &buf,
		Level: logrus.ErrorLevel,
		Formatter: &logrus.TextFormatter{
			DisableColors:    true,
			DisableTimestamp: true,
			DisableSorting:   true,
		},
	}
	out, err := New(dir, size, testLogger.WithField("cache", "test"))
	assert.NoError(t, err)
	cleanup := func() {
		out.Close()
		assert.Equal(t, -1, strings.LastIndex(buf.String(), "closing cache"))
		assert.NoError(t, os.RemoveAll(dir))
	}
	return out, &buf, cleanup
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		size += fi.Size()
		return err
	})
	return size, err
}
