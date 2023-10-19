// Copyright (C) 2023 Percona LLC
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

// Package dump wraps pmm-dump integration.
package dump

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

var ErrDumpAlreadyRunning = status.Error(codes.FailedPrecondition, "pmm-dump already running.")

const (
	pmmDumpBin = "pmm-dump"
	dumpsDir   = "/srv/dump"
)

type Service struct {
	l *logrus.Entry

	db *reform.DB

	running atomic.Bool

	rw     sync.RWMutex
	cancel context.CancelFunc
}

func New(db *reform.DB) *Service {
	return &Service{
		l:  logrus.WithField("component", "management/backup/backup"),
		db: db,
	}
}

type Params struct {
	APIKey       string
	ServiceNames []string
	StartTime    time.Time
	EndTime      time.Time
	ExportQAN    bool
	IgnoreLoad   bool
}

func (s *Service) StartDump(params *Params) (string, error) {
	// Check is some dump already running
	if !s.running.CompareAndSwap(false, true) {
		return "", ErrDumpAlreadyRunning
	}

	dump, err := models.CreateDump(s.db.Querier, models.CreateDumpParams{
		ServiceNames: params.ServiceNames,
		StartTime:    params.StartTime,
		EndTime:      params.EndTime,
		ExportQAN:    params.ExportQAN,
		IgnoreLoad:   params.IgnoreLoad,
	})
	if err != nil {
		s.running.Store(false)
		return "", errors.Wrap(err, "failed to create dump")
	}

	ctx, cancel := context.WithCancel(context.Background())

	s.rw.Lock()
	s.cancel = cancel
	s.rw.Unlock()

	pmmDumpCmd := exec.CommandContext(ctx,
		pmmDumpBin,
		"export",
		"--pmm-url=http://127.0.0.1",
		fmt.Sprintf(`--pmm-token=%s`, params.APIKey),
		fmt.Sprintf("--dump-path=%s", getDumpPath(dump.ID)))

	for _, serviceName := range params.ServiceNames {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf("--instance=%s", serviceName))
	}

	if !params.StartTime.IsZero() {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf("--start-ts=%s", params.StartTime.Format(time.RFC3339)))
	}

	if !params.EndTime.IsZero() {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf("--end-ts=%s", params.EndTime.Format(time.RFC3339)))
	}

	if params.ExportQAN {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, "--dump-qan")
	}

	if params.IgnoreLoad {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, "--ignore-load")
	}

	pReader, pWriter := io.Pipe()
	pmmDumpCmd.Stdout = pWriter
	pmmDumpCmd.Stderr = pWriter

	go func() {
		defer pReader.Close() //nolint:errcheck

		err := s.persistLogs(ctx, dump.ID, pReader)
		if err != nil && !errors.Is(err, context.Canceled) {
			s.l.Errorf("pmm-dump logs persist failed: %v", err)
		}
	}()

	go func() {
		// Switch running flag back to false
		defer s.running.Store(false)
		defer s.cancel()
		defer pWriter.Close() //nolint:errcheck

		err := pmmDumpCmd.Run()
		if err != nil {
			s.l.Errorf("failed to execute pmm-dump: %v", err)

			s.setDumpStatus(dump.ID, models.DumpStatusError)
			return
		}

		s.setDumpStatus(dump.ID, models.DumpStatusSuccess)
	}()

	return dump.ID, nil
}

func (s *Service) DeleteDump(dumpID string) error {
	path := getDumpPath(dumpID)
	err := os.Remove(path)
	if err != nil {
		return errors.Wrap(err, "failed to remove pmm-dump files")
	}

	if err = models.DeleteDump(s.db.Querier, dumpID); err != nil {
		return errors.Wrap(err, "failed to delete dump")
	}

	return nil
}

func (s *Service) setDumpStatus(dumpID string, status models.DumpStatus) {
	if err := s.db.InTransaction(func(t *reform.TX) error {
		return models.UpdateDumpStatus(t.Querier, dumpID, status)
	}); err != nil {
		s.l.Warnf("Failed to update dupm status: %+v", err)
	}
}

func (s *Service) persistLogs(ctx context.Context, dumpID string, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	var err error
	var chunkN uint32

ScanLoop:
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.l.Warnf("Failed to read pmm-dump logs: %+v", ctx.Err())
			nErr := s.saveLogChunk(dumpID, atomic.AddUint32(&chunkN, 1)-1, ctx.Err().Error(), true)
			if nErr != nil {
				return errors.WithStack(nErr)
			}

			break ScanLoop
		default:
			// continue
		}

		nErr := s.saveLogChunk(dumpID, atomic.AddUint32(&chunkN, 1)-1, scanner.Text(), false)
		if nErr != nil {
			s.l.Warnf("failed to read pmm-dump logs: %v", err)
			return errors.WithStack(nErr)
		}
	}

	if err = scanner.Err(); err != nil {
		s.l.Warnf("Failed to read pmm-dump logs: %+v", err)
		nErr := s.saveLogChunk(dumpID, atomic.AddUint32(&chunkN, 1)-1, err.Error(), false)
		if nErr != nil {
			return errors.WithStack(nErr)
		}
	}

	nErr := s.saveLogChunk(dumpID, atomic.AddUint32(&chunkN, 1)-1, "", true)
	if nErr != nil {
		return errors.WithStack(nErr)
	}

	return nil
}

func (s *Service) saveLogChunk(dumpID string, chunkN uint32, text string, last bool) error {
	if _, err := models.CreateDumpLog(s.db.Querier, models.CreateDumpLogParams{
		DumpID:    dumpID,
		ChunkID:   atomic.AddUint32(&chunkN, 1) - 1,
		Data:      text,
		LastChunk: last,
	}); err != nil {
		return errors.Wrap(err, "failed to save pmm-dump log chunk")
	}

	return nil
}

func (s *Service) StopDump() {
	s.rw.RLock()
	defer s.rw.RUnlock()

	s.cancel()
}

func getDumpPath(id string) string {
	return fmt.Sprintf("%s/%s.tar.gz", dumpsDir, id)
}
