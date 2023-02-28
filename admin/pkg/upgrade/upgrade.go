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

// Package upgrade holds logic for upgrading PMM Server.
package upgrade

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/cli/flags"
	dockerCmd "github.com/percona/pmm/admin/commands/pmm/server/docker"
)

const logsFileNamePattern = "upgrade.*.log"

// StatusResponse holds information about upgrade status.
type StatusResponse struct {
	// Individual log lines.
	Lines []string
	// Offset for the next log line.
	Offset uint32
	// True when upgrade has finished.
	Done bool
}

// Upgrader manages PMM Server upgrades.
type Upgrader struct {
	docker                 containerManager
	dockerImage            string
	newContainerNamePrefix string

	gRPCMessageMaxSize uint32

	upgradeInProgress map[string]struct{}
	upgradeMu         sync.RWMutex
}

// New returns new Upgrader.
func New(docker containerManager, dockerImage, newContainerNamePrefix string, gRPCMessageMaxSize uint32) *Upgrader {
	return &Upgrader{
		docker:                 docker,
		dockerImage:            dockerImage,
		gRPCMessageMaxSize:     gRPCMessageMaxSize,
		newContainerNamePrefix: newContainerNamePrefix,
		upgradeInProgress:      make(map[string]struct{}, 1),
	}
}

// StartUpgrade starts PMM Server upgrade.
func (u *Upgrader) StartUpgrade(ctx context.Context, containerID string) (string, error) {
	logrus.Debugf("Inspecting container %s", containerID)
	container, err := u.docker.ContainerInspect(ctx, containerID)
	if err != nil {
		logrus.Errorf("Could not inspect container %s. Error: %v", containerID, err)
		return "", err
	}

	if !container.State.Running {
		return "", fmt.Errorf("container %s is not running", containerID)
	}

	logFile, err := os.CreateTemp("", logsFileNamePattern)
	if err != nil {
		return "", err
	}

	go func() {
		logrus.Infof("Starting upgrade of container %s with log file %s", containerID, logFile.Name())
		defer func() {
			// Keep the log around for a short period of time
			<-time.After(5 * time.Minute)
			if err := os.Remove(logFile.Name()); err != nil {
				logrus.Error(err)
			}
		}()

		logger := logrus.New()
		logger.SetOutput(io.MultiWriter(logFile, os.Stdout))

		newContainerNamePrefix := u.newContainerNamePrefix
		if newContainerNamePrefix == "" {
			newContainerNamePrefix = "pmm-server"
		}

		cmd := dockerCmd.NewUpgradeCommand(
			logger.WithField("upgrade", logFile.Name()),
			5*time.Second)
		cmd.ContainerID = containerID
		cmd.DockerImage = u.dockerImage
		cmd.NewContainerNamePrefix = newContainerNamePrefix
		cmd.AssumeYes = true

		// Store upgrade in progress info.
		u.upgradeMu.Lock()
		u.upgradeInProgress[logFile.Name()] = struct{}{}
		u.upgradeMu.Unlock()

		defer func() {
			u.upgradeMu.Lock()
			defer u.upgradeMu.Unlock()

			delete(u.upgradeInProgress, logFile.Name())
		}()

		_, err := cmd.RunCmdWithContext(ctx, &flags.GlobalFlags{})
		if err != nil {
			logger.Errorf("Could not upgrade container %s. Error: %v", containerID, err)
		}
	}()

	return logFile.Name(), nil
}

// UpgradeStatus returns PMM Server upgrade status.
func (u *Upgrader) UpgradeStatus(ctx context.Context, logsToken string, offset uint32) *StatusResponse {
	var err error
	var lines []string
	var newOffset uint32
	var done bool

	// wait up to 30 seconds for new log lines
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		done = !u.isUpgradeRunning(logsToken)
		if done {
			// give a second to flush logs to file
			time.Sleep(time.Second)
		}

		lines, newOffset, err = u.getLogs(logsToken, offset)
		if err != nil {
			logrus.Warn(err)
		}

		if len(lines) != 0 || done {
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	if err = ctx.Err(); !errors.Is(err, context.DeadlineExceeded) {
		logrus.Warnf("context error during UpgradeStatus: %s", err)
	}

	return &StatusResponse{
		Lines:  lines,
		Offset: newOffset,
		Done:   done,
	}
}

func (u *Upgrader) isUpgradeRunning(name string) bool {
	u.upgradeMu.RLock()
	defer u.upgradeMu.RUnlock()

	_, ok := u.upgradeInProgress[name]

	return ok
}

// IsAnyUpgradeRunning returns true if there's at least one upgrade in progress.
func (u *Upgrader) IsAnyUpgradeRunning() bool {
	u.upgradeMu.RLock()
	defer u.upgradeMu.RUnlock()

	return len(u.upgradeInProgress) != 0
}

// getLogs returns some lines and a new offset from a log file starting from the given offset.
// It may return zero lines and the same offset. Caller is expected to handle this.
func (u *Upgrader) getLogs(filePath string, offset uint32) ([]string, uint32, error) {
	if err := u.isValidLogPath(filePath); err != nil {
		return nil, 0, errors.WithStack(err)
	}

	f, err := os.Open(filePath) //nolint:gosec
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	defer f.Close() //nolint:gosec

	if _, err = f.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, 0, errors.WithStack(err)
	}

	lines := make([]string, 0, 10)
	reader := bufio.NewReader(f)
	newOffset := offset
	for {
		line, err := reader.ReadString('\n')
		if err == nil {
			newOffset += uint32(len(line))
			if newOffset-offset > u.gRPCMessageMaxSize {
				return lines, newOffset - uint32(len(line)), nil
			}
			lines = append(lines, strings.TrimSuffix(line, "\n"))
			continue
		}
		if err == io.EOF {
			err = nil
		}
		return lines, newOffset, errors.WithStack(err)
	}
}

func (u *Upgrader) isValidLogPath(filePath string) error {
	filename := path.Base(filePath)
	match, err := path.Match(logsFileNamePattern, filename)
	if err != nil {
		return err
	}

	if !match {
		return fmt.Errorf("invalid log file path provided")
	}

	return nil
}
