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

// Package wal provides a durable, size-bounded on-disk queue (write-ahead log)
// for QAN metrics buckets so collected data survives transient send failures and
// agent restarts. Delivery is at-least-once: an entry is removed only after the
// server acknowledges it, and the receiver deduplicates by idempotency key.
package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// DirName is the write-ahead log subdirectory created under the agent's base path.
const DirName = "wal"

const (
	fileExt = ".pb"
	tmpExt  = ".tmp"

	dirPerm  = 0o700
	filePerm = 0o600
)

type entry struct {
	seq  uint64
	size int64
}

// Item is a request dequeued from the spool, tagged with its sequence number.
type Item struct {
	Seq     uint64
	Request *agentv1.QANCollectRequest
}

// Spool is a durable, size-bounded on-disk FIFO of QANCollectRequests. When the
// configured size cap is exceeded the oldest entries are dropped. It is safe for
// one concurrent producer (Enqueue) and one consumer (Next/Remove).
type Spool struct {
	dir      string
	maxBytes int64
	l        *logrus.Entry
	notify   chan struct{}

	mu        sync.Mutex
	seq       uint64
	entries   []entry
	totalSize int64
	dropped   uint64
}

// New opens (creating if needed) a spool in dir, replaying entries left by a
// previous run so that no collected-but-unsent data is lost across restarts.
func New(dir string, maxBytes int64, l *logrus.Entry) (*Spool, error) {
	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return nil, fmt.Errorf("create wal dir: %w", err)
	}

	s := &Spool{
		dir:      dir,
		maxBytes: maxBytes,
		l:        l,
		notify:   make(chan struct{}, 1),
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read wal dir: %w", err)
	}
	for _, de := range dirEntries {
		name := de.Name()
		if de.IsDir() {
			continue
		}
		if strings.HasSuffix(name, tmpExt) {
			// Leftover from a write interrupted by a crash.
			rerr := os.Remove(filepath.Join(dir, name))
			if rerr != nil {
				l.Warnf("Failed to remove stale wal temp file %s: %s", name, rerr)
			}
			continue
		}
		if !strings.HasSuffix(name, fileExt) {
			continue
		}
		seq, perr := strconv.ParseUint(strings.TrimSuffix(name, fileExt), 10, 64)
		if perr != nil {
			continue
		}
		info, ierr := de.Info()
		if ierr != nil {
			continue
		}
		s.entries = append(s.entries, entry{seq: seq, size: info.Size()})
		s.totalSize += info.Size()
		if seq >= s.seq {
			s.seq = seq + 1
		}
	}
	sort.Slice(s.entries, func(i, j int) bool { return s.entries[i].seq < s.entries[j].seq })
	// No concurrency before New returns; trim a recovered over-capacity spool.
	s.removeFiles(s.takeOverCapLocked())

	if n := len(s.entries); n > 0 {
		l.Infof("Recovered %d spooled QAN requests (%d bytes) from %s.", n, s.totalSize, dir)
		s.signal()
	}
	return s, nil
}

// Enqueue durably appends req to the spool, dropping the oldest entries if the
// size cap would be exceeded.
func (s *Spool) Enqueue(req *agentv1.QANCollectRequest) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal QAN request: %w", err)
	}

	// seq is owned by the single producer goroutine; no other goroutine touches it.
	seq := s.seq
	s.seq++

	// Write the file outside the lock: it is not referenced by s.entries yet, so
	// the consumer cannot observe it and a slow fsync must not block Next/Remove.
	tmp := filepath.Join(s.dir, fmt.Sprintf("%020d%s", seq, tmpExt))
	err = writeFileSync(tmp, data)
	if err != nil {
		return err
	}
	err = os.Rename(tmp, s.fileName(seq))
	if err != nil {
		return fmt.Errorf("rename wal file: %w", err)
	}
	s.syncDir()

	// Publish the entry and pick over-capacity victims under the lock (in-memory
	// only); delete the victim files afterwards, off the lock.
	s.mu.Lock()
	s.entries = append(s.entries, entry{seq: seq, size: int64(len(data))})
	s.totalSize += int64(len(data))
	victims := s.takeOverCapLocked()
	s.signal()
	s.mu.Unlock()

	s.removeFiles(victims)
	return nil
}

// Next returns the oldest entry without removing it, skipping (and discarding)
// any unreadable entries. The second result is false when the spool is empty.
func (s *Spool) Next() (*Item, bool) {
	for {
		s.mu.Lock()
		if len(s.entries) == 0 {
			s.mu.Unlock()
			return nil, false
		}
		seq := s.entries[0].seq
		s.mu.Unlock()

		// Read the file outside the lock. If it was evicted concurrently (producer
		// over-capacity drop), the read fails and we skip to the next entry.
		data, err := os.ReadFile(s.fileName(seq)) //nolint:gosec
		if err != nil {
			s.l.Warnf("Failed to read spooled QAN request %d, dropping: %s", seq, err)
			s.removeBySeq(seq)
			continue
		}
		req := &agentv1.QANCollectRequest{}
		err = proto.Unmarshal(data, req)
		if err != nil {
			s.l.Warnf("Failed to unmarshal spooled QAN request %d, dropping: %s", seq, err)
			s.removeBySeq(seq)
			continue
		}
		return &Item{Seq: seq, Request: req}, true
	}
}

// Remove discards the entry with the given sequence number after it has been
// acknowledged by the server. It is a no-op if the entry is already gone.
func (s *Spool) Remove(seq uint64) {
	s.removeBySeq(seq)
}

// Notify returns a channel that receives a value whenever an entry is enqueued,
// so a consumer parked on an empty spool can wake up.
func (s *Spool) Notify() <-chan struct{} { return s.notify }

// Len returns the number of spooled entries.
func (s *Spool) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

// Dropped returns the number of entries dropped due to the size cap.
func (s *Spool) Dropped() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func (s *Spool) fileName(seq uint64) string {
	return filepath.Join(s.dir, fmt.Sprintf("%020d%s", seq, fileExt))
}

// takeOverCapLocked evicts the oldest entries while over the size cap and returns
// their sequence numbers so the caller can delete the files outside the lock.
// The caller must hold s.mu.
func (s *Spool) takeOverCapLocked() []uint64 {
	var victims []uint64
	for s.totalSize > s.maxBytes && len(s.entries) > 1 {
		v := s.entries[0]
		s.entries = s.entries[1:]
		s.totalSize -= v.size
		s.dropped++
		victims = append(victims, v.seq)
		s.l.Warnf("QAN wal over capacity (%d bytes); dropped oldest request %d. Total dropped: %d.",
			s.maxBytes, v.seq, s.dropped)
	}
	return victims
}

// removeBySeq drops the entry with the given sequence number from the in-memory
// index (under the lock) and deletes its file (outside the lock).
func (s *Spool) removeBySeq(seq uint64) {
	s.mu.Lock()
	found := false
	for i, e := range s.entries {
		if e.seq == seq {
			s.totalSize -= e.size
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			found = true
			break
		}
	}
	s.mu.Unlock()
	if found {
		s.removeFiles([]uint64{seq})
	}
}

// removeFiles deletes spool files by sequence number. Safe to call without the lock.
func (s *Spool) removeFiles(seqs []uint64) {
	for _, seq := range seqs {
		err := os.Remove(s.fileName(seq))
		if err != nil && !os.IsNotExist(err) {
			s.l.Warnf("Failed to remove spooled QAN request %d: %s", seq, err)
		}
	}
}

func (s *Spool) signal() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

// writeFileSync writes data to path and fsyncs it before returning.
func writeFileSync(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm) //nolint:gosec
	if err != nil {
		return fmt.Errorf("create wal file: %w", err)
	}

	_, err = f.Write(data)
	if err != nil {
		closeQuietly(f)
		return fmt.Errorf("write wal file: %w", err)
	}
	err = f.Sync()
	if err != nil {
		closeQuietly(f)
		return fmt.Errorf("sync wal file: %w", err)
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("close wal file: %w", err)
	}
	return nil
}

func closeQuietly(f *os.File) {
	_ = f.Close() //nolint:gosec
}

// syncDir fsyncs the spool directory so a rename is durable. Best-effort: some
// platforms do not support directory fsync, so a failure is logged, not fatal
// (the file's own contents are already fsynced before the rename).
func (s *Spool) syncDir() {
	d, err := os.Open(s.dir) //nolint:gosec
	if err != nil {
		s.l.Warnf("Failed to open wal dir for sync: %s", err)
		return
	}
	err = d.Sync()
	if err != nil {
		s.l.Warnf("Failed to fsync wal dir: %s", err)
	}
	cerr := d.Close()
	if cerr != nil {
		s.l.Warnf("Failed to close wal dir: %s", cerr)
	}
}
