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
	"path/filepath"
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

// ErrDumpAlreadyRunning is an exported error indicating that pmm-dump is already running.
var ErrDumpAlreadyRunning = status.Error(codes.FailedPrecondition, "pmm-dump already running.")

const (
	pmmDumpBin = "pmm-dump"
	dumpsDir   = "/srv/dump"
)

// Service represents the dump service.
type Service struct {
	l *logrus.Entry

	db *reform.DB

	running atomic.Bool

	rw     sync.RWMutex
	cancel context.CancelFunc
}

// New creates a new instance of the dump service..
func New(db *reform.DB) *Service {
	return &Service{
		l:  logrus.WithField("component", "management/backup/backup"),
		db: db,
	}
}

// Params represents the parameters for configuring the dump service.
type Params struct {
	APIKey       string
	Cookie       string
	User         string
	Password     string
	ServiceNames []string
	StartTime    *time.Time
	EndTime      *time.Time
	ExportQAN    bool
	IgnoreLoad   bool
}

// StartDump initiates the process of creating and managing dumps in the dump service.
func (s *Service) StartDump(params *Params) (string, error) {
	// Check if some pmm-dump already running.
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

	l := s.l.WithField("dump_id", dump.ID)

	ctx, cancel := context.WithCancel(context.Background())

	s.rw.Lock()
	s.cancel = cancel
	s.rw.Unlock()

	pmmDumpCmd := exec.CommandContext(ctx, //nolint:gosec
		pmmDumpBin,
		"export",
		"--pmm-url=http://127.0.0.1",
		fmt.Sprintf("--dump-path=%s", getDumpFilePath(dump.ID)))

	if params.APIKey != "" {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf(`--pmm-token=%s`, params.APIKey))
	}

	if params.Cookie != "" {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf(`--pmm-cookie=%s`, params.Cookie))
	}

	if params.User != "" {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf(`--pmm-user=%s`, params.User))
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf(`--pmm-pass=%s`, params.Password))
	}

	for _, serviceName := range params.ServiceNames {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf("--instance=%s", serviceName))
	}

	if params.StartTime != nil {
		pmmDumpCmd.Args = append(pmmDumpCmd.Args, fmt.Sprintf("--start-ts=%s", params.StartTime.Format(time.RFC3339)))
	}

	if params.EndTime != nil {
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

		err := s.persistLogs(dump.ID, pReader)
		if err != nil && !errors.Is(err, context.Canceled) {
			l.Errorf("Dump logs persisting failed: %v", err)
		}

		l.Info("Dump logs saved.")
	}()

	go func() {
		// Switch running flag back to false
		defer s.running.Store(false)
		defer s.cancel()
		defer pWriter.Close() //nolint:errcheck

		err := pmmDumpCmd.Run()
		if err != nil {
			l.Errorf("Failed to execute pmm-dump: %v", err)

			s.setDumpStatus(dump.ID, models.DumpStatusError)
			return
		}

		s.setDumpStatus(dump.ID, models.DumpStatusSuccess)
		l.WithField("dump_id", dump.ID).Info("Dump done.")
	}()

	return dump.ID, nil
}

// DeleteDump removes a specific dump associated with the dump service.
func (s *Service) DeleteDump(dumpID string) error {
	dump, err := models.FindDumpByID(s.db.Querier, dumpID)
	if err != nil {
		return errors.Wrap(err, "failed to find dump")
	}

	filePath := getDumpFilePath(dump.ID)
	err = validateFilePath(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.WithStack(err)
	}

	err = os.Remove(filePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "failed to remove pmm-dump files")
	}

	if err = models.DeleteDump(s.db.Querier, dumpID); err != nil {
		return errors.Wrap(err, "failed to delete dump")
	}

	return nil
}

// GetFilePathsForDumps retrieves the file paths associated with the dumps managed by the dump service.
func (s *Service) GetFilePathsForDumps(dumpIDs []string) (map[string]string, error) {
	dumps, err := models.FindDumpsByIDs(s.db.Querier, dumpIDs)
	if err != nil {
		return nil, err
	}

	res := make(map[string]string, len(dumps))
	for _, d := range dumps {
		if d.Status != models.DumpStatusSuccess {
			s.l.Warnf("Dump with id %s is in %s state. Skiping it.", d.ID, d.Status)
			continue
		}
		filePath := getDumpFilePath(d.ID)
		if err = validateFilePath(filePath); err != nil {
			return nil, errors.WithStack(err)
		}

		res[d.ID] = filePath
	}

	return res, nil
}

func (s *Service) setDumpStatus(dumpID string, status models.DumpStatus) {
	if err := s.db.InTransaction(func(t *reform.TX) error {
		return models.UpdateDumpStatus(t.Querier, dumpID, status)
	}); err != nil {
		s.l.Warnf("Failed to update dupm status: %+v", err)
	}
}

func (s *Service) persistLogs(dumpID string, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	var err error
	var chunkN uint32

	for scanner.Scan() {
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

// StopDump stops the ongoing dump process in the dump service.
func (s *Service) StopDump() {
	s.rw.RLock()
	defer s.rw.RUnlock()

	s.cancel()
}

func getDumpFilePath(id string) string {
	return fmt.Sprintf("%s/%s.tar.gz", dumpsDir, id)
}

func validateFilePath(path string) error {
	c := filepath.Clean(path)
	r, err := filepath.EvalSymlinks(c)
	if err != nil {
		return errors.Wrap(err, "unsafe or invalid dump filepath")
	}

	if path != r {
		return errors.Errorf("actual file path doesn't match expected, that may be caused by symlinks "+
			"of path traversal, expected path: %s, actual: %s", path, r)
	}

	return nil
}
